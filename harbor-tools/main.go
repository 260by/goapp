// 显示, 获取, 提交harbor私有仓库中base项目下的docker镜像
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
		repo, err := getRepository(user, passwd, baseURL, project)
		if err != nil {
			panic(err)
		}

		for _, v := range repo {
			// tags大于100时进行删除镜像
			if v.TagsCount > 100 {
				tags, err := getTags(user, passwd, baseURL, v.RepositoryName)
				if err != nil {
					panic(err)
				}

				// 保留50个最近的镜像
				j := len(tags) - 50
				for i := 0; i < j; i++ {
					if err := deleteDockerImage(user, passwd, baseURL, v.RepositoryName, tags[i].Name); err != nil {
						log.Printf("delete %s:%s error: %v", v.RepositoryName, tags[i].Name, err)
						continue
					}
					log.Printf("delete %s:%s", v.RepositoryName, tags[i].Name)
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
