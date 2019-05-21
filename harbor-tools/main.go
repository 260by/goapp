// 显示, 获取, 提交harbor私有仓库中base项目下的docker镜像
package main

import (
	"flag"
	"fmt"
	"net/http"
	"io/ioutil"
	"os/exec"
	// "sync"
	"strings"
	"time"
	"github.com/tidwall/gjson"
)

var (
	baseURL string
	user string
	passwd string
	show bool
	pull bool
	push bool
	imagesFile string
	project string
)

func usage() {
	fmt.Printf(`Description: show、pull、push harbor docker registry under the appoint project images. Executed show is and saved to a file at the same time, Ex: /tmp/docker-images-201811221805.txt
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
	flag.StringVar(&imagesFile, "file", "", "Docker image list file name")
	flag.StringVar(&project, "project", "", "Harbor registry project name")
	flag.Usage = usage
	flag.Parse()

	if baseURL == "" || user == "" || passwd == "" || project == "" {
		flag.Usage()
		return
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