// 显示, 获取, 提交harbor私有仓库中base项目下的docker镜像
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	// "log"
	"net/http"
	"os"
	"os/exec"
	"sort"

	// "sync"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var (
	baseURL    string
	user       string
	passwd     string
	show       bool
	pull       bool
	push       bool
	del        bool
	imagesFile string
	project    string
	// 阿里云老集群
	aliyunK8sAddr = ""
	aliyunK8sToken = ""
	// 阿里云新集群
	newAliyunK8sAddr = ""
	newAliyunK8sToken = ""
	// AWS东京
	tokyoK8sAddr = ""
	tokyoK8sToken = ""
)

func usage() {
	fmt.Printf(`Description: show、pull、push、delete harbor docker registry under the appoint project images. Executed show is and saved to a file at the same time, Ex: /tmp/docker-images-201811221805.txt
Options:
`)
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&baseURL, "url", "", "harbor docker registry address, Ex: reg.harbor.com")
	flag.StringVar(&user, "user", "", "harbor registry user name")
	flag.StringVar(&passwd, "password", "", "harbor registry user password")
	flag.BoolVar(&show, "show", false, "Show docker images")
	flag.BoolVar(&pull, "pull", false, "Pull docker images to local")
	flag.BoolVar(&push, "push", false, "Push docker images to registry")
	flag.BoolVar(&del, "del", false, "Delete docker images, default retain lately 50")
	flag.StringVar(&imagesFile, "file", "", "Docker image list file name")
	flag.StringVar(&project, "project", "", "Harbor registry project name")
	flag.Usage = usage
	flag.Parse()

	if baseURL == "" || user == "" || passwd == "" || project == "" {
		flag.Usage()
		return
	}

	if del {
		var images []string
		aliyunStableImages, err := getK8sPodsImage(aliyunK8sAddr, "stable", aliyunK8sToken)
		if err != nil {
			log.Printf("aliyun old k8s stable %v", err)
		}
		aliyunAuditImages, err := getK8sPodsImage(aliyunK8sAddr, "audit", aliyunK8sToken)
		if err != nil {
			log.Printf("aliyun old k8s audit %v", err)
		}

		newAliyunStableImages, err := getK8sPodsImage(aliyunK8sAddr, "stable", aliyunK8sToken)
		if err != nil {
			panic(err)
		}
		newAliyunAuditImages, err := getK8sPodsImage(aliyunK8sAddr, "audit", aliyunK8sToken)
		if err != nil {
			panic(err)
		}

		tokyoStableImages, err := getK8sPodsImage(aliyunK8sAddr, "stable", aliyunK8sToken)
		if err != nil {
			panic(err)
		}
		tokyoAuditImages, err := getK8sPodsImage(aliyunK8sAddr, "audit", aliyunK8sToken)
		if err != nil {
			panic(err)
		}

		images = append(images, aliyunStableImages...)
		images = append(images, aliyunAuditImages...)
		images = append(images, newAliyunStableImages...)
		images = append(images, newAliyunAuditImages...)
		images = append(images, tokyoStableImages...)
		images = append(images, tokyoAuditImages...)
	
		

		repo, err := getRepository(user, passwd, baseURL, project)
		if err != nil {
			panic(err)
		}

		for _, v := range repo {
			// tags大于60时进行删除镜像
			if v.TagsCount > 100 {
				tags, err := getTags(user, passwd, baseURL, v.RepositoryName)
				if err != nil {
					panic(err)
				}

				for _, tag := range tags {
					if IsExistItem(fmt.Sprintf("%s:%s", v.RepositoryName, tag.Name), images) {
						log.Printf("%s:%s 镜像正式和审核环境正在使用", v.RepositoryName, tag.Name)
						continue
					}

					log.Printf("%s:%s 可以删除", v.RepositoryName, tag.Name)
					// if err := deleteDockerImage(user, passwd, baseURL, v.RepositoryName, tag.Name); err != nil {
					// 	log.Printf("delete %s:%s error: %v", v.RepositoryName, tag.Name, err)
					// 	continue
					// }
				}
			}
		}
		os.Exit(0)
	}

	images := getImage()

	if show {
		var filename, fileContent string
		filename = fmt.Sprintf("/tmp/docker-images-%s.txt", time.Now().Format("200601021504"))
		for k := range images {
			fmt.Println(images[k])
			fileContent += fmt.Sprintf("%s\n", images[k])
		}
		if err := ioutil.WriteFile(filename, []byte(fileContent), 0644); err != nil {
			panic(err)
		} else {
			fmt.Printf("\033[1;32mOutput to %s\033[0m\n", filename)
		}
	}

	if pull {
		// var wg sync.WaitGroup
		for _, image := range images {
			// wg.Add(1)
			// go func(image string)  {
			// 	defer wg.Add(-1)
			// 	cmd := fmt.Sprintf("docker pull %s", image)
			// 	_, err := exec.Command("sh", "-c", cmd).Output()
			// 	if err != nil {
			// 		panic(err)
			// 	} else {
			// 		fmt.Printf("Pull %s image is success.\n", image)
			// 	}
			// }(image)

			cmd := fmt.Sprintf("docker pull %s", image)
			_, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("Pull %s image is success.\n", image)
			}
		}
		// wg.Wait()
	}

	if push && imagesFile != "" {
		f, err := ioutil.ReadFile(imagesFile)
		if err != nil {
			panic(err)
		}
		imageList := strings.Split(strings.TrimSuffix(string(f), "\n"), "\n")

		for k := range imageList {
			cmd := fmt.Sprintf("docker push %s", imageList[k])
			_, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("Push %s image is success.\n", imageList[k])
			}
		}
	}

}

func getImage() (images []string) {
	client := &http.Client{}

	repoURL := fmt.Sprintf("https://%s:%s@%s/api/search?q=%s", user, passwd, baseURL, project)
	tagBaseURL := fmt.Sprintf("https://%s:%s@%s/api/repositories/", user, passwd, baseURL)
	req, err := http.NewRequest("GET", repoURL, nil)

	req.Header.Add("Content-Type", "application/json")

	if err != nil {
		panic(err)
	}

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	repositoryNames := gjson.Get(string(body), "repository.#.repository_name")

	for _, repo := range repositoryNames.Array() {
		getTagsURL := fmt.Sprintf("%s%s/tags", tagBaseURL, repo)
		reqTags, err := http.NewRequest("GET", getTagsURL, nil)
		reqTags.Header.Add("Content-Type", "application/json")
		if err != nil {
			panic(err)
		}
		resTags, err := client.Do(reqTags)
		if err != nil {
			panic(err)
		}
		defer resTags.Body.Close()
		tagsBody, err := ioutil.ReadAll(resTags.Body)
		if err != nil {
			panic(err)
		}

		tags := gjson.Get(string(tagsBody), "#.name")

		for _, tag := range tags.Array() {
			images = append(images, fmt.Sprintf("%s/%s:%s", baseURL, repo, tag))
		}
	}
	return
}

// Repository 存储库
type Repository struct {
	ProjectID      int    `json:"project_id"`
	ProjectName    string `json:"project_name"`
	ProjectPublic  bool   `json:"project_public"`
	PullCount      int    `json:"pull_count"`
	RepositoryName string `json:"repository_name"`
	TagsCount      int    `json:"tags_count"`
}

// getRepository 获取项目下的存储库
func getRepository(user, password, addr, project string) ([]Repository, error) {
	url := fmt.Sprintf("https://%s:%s@%s/api/search?q=%s", user, password, addr, project)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	s := gjson.GetBytes(body, "repository").String()

	repo := []Repository{}
	if err := json.Unmarshal([]byte(s), &repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// Tag tag
type Tag struct {
	Digest        string `json:"digest"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	Architecture  string `json:"architecture"`
	OS            string `json:"os"`
	OSVersion     string `json:"os.version"`
	DockerVersion string `json:"docker_version"`
	Author        string `json:"author"`
	Created       string `json:"created"`
}

// getTags 获取存储库tag列表
func getTags(user, password, addr, repository string) ([]Tag, error) {
	Loop:
	url := fmt.Sprintf("https://%s:%s@%s/api/repositories/%s/tags", user, password, addr, repository)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 504 {
		goto Loop
	}
	
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("get tages error, status code %v", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tags := []Tag{}
	if err := json.Unmarshal(body, &tags); err != nil {
		return nil, err
	}

	// 升序
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Created < tags[j].Created
	})

	return tags, nil
}

// deleteDockerImage 删除docker镜像
func deleteDockerImage(user, password, addr, repository, tag string) error {
	url := fmt.Sprintf("https://%s:%s@%s/api/repositories/%s/tags/%s", user, password, addr, repository, tag)
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// K8SRequest k8s http request
func K8SRequest(method, url, content, token string) (int, []byte, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(method, url, strings.NewReader(content))
	if err != nil {
			return 500, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/yaml")

	res, err := client.Do(req)
	if err != nil {
			return 500, nil, err
	}
	defer res.Body.Close()

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
			return 500, nil, err
	}

	return res.StatusCode, response, nil
}

func getK8sPodsImage(k8sAddr, namespace, token string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", k8sAddr, namespace)
	code, resp, err := K8SRequest("GET", url, "", token)
	if err != nil {
		return nil, err
	}

	if code != 200 {
		return nil, fmt.Errorf("request code %v", code)
	}

	var images []string
	items := gjson.GetBytes(resp, "items").Array()
	for _, item := range items {
		containers := gjson.Get(item.String(), "spec.containers").Array()
		for _, container := range containers {
			image := gjson.Get(container.String(), "image").String()
			images = append(images, strings.TrimPrefix(image, "reg.haochang.tv/"))
		}
	}

	return removeDuplicates(images), nil
}

func removeDuplicates(elements []string) []string {
    // Use map to record duplicates as we find them.
    encountered := map[string]bool{}
    result := []string{}
    for v := range elements {
        if encountered[elements[v]] == true {
            // Do not add duplicate.
        } else {
            // Record this element as an encountered element.
            encountered[elements[v]] = true
            // Append to result slice.
            result = append(result, elements[v])
        }
    }
    // Return the new slice.
    return result
}

// IsExistItem 判断slice中是否存在某个item
func IsExistItem(value interface{}, array interface{}) bool {
    switch reflect.TypeOf(array).Kind() {
    case reflect.Slice:
        s := reflect.ValueOf(array)
        for i := 0; i < s.Len(); i++ {
            if reflect.DeepEqual(value, s.Index(i).Interface()) {
                return true
            }
        }
    }
    return false
}