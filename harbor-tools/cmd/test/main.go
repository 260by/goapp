package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	token = ""
	namespace = "appos-rc"
)

func main()  {
	images, err := getK8sImages("https://10.0.0.110:8443", namespace)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(images))
	fmt.Println(images)
}

func getK8sImages(addr, namespaces string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", addr, namespace)
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
