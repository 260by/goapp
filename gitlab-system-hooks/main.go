package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	// "log"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"github.com/tidwall/gjson"
)

// Project 项目信息
type Project struct {
	ID	 string
	Name string
	Path string
	PathWithNamespace string
	Language string
	Env []string
}

// CreateFileOptions 创建文件选项
type CreateFileOptions struct {
	Branch        string `json:"branch,omitempty"`  		// 分支名称
	AuthorEmail   string `json:"author_email,omitempty"` 	// 提交者Email 
	AuthorName    string `json:"author_name,omitempty"`		// 提交者
	Actions       []Action `json:"actions,omitempty"`		// 动作
	CommitMessage string `json:"commit_message,omitempty"`	// 提交消息
}

// Action 动作
type Action struct {
	Action string `json:"action,omitempty"`			// 动作，包含create,delete,move,update,chmod
	FilePath string `json:"file_path,omitempty"`	// 提交文件的完整路径. Ex app/main.go
	Content string `json:"content,omitempty"`		// 文件内容
	Encoding string `json:"encoding,omitempty"`		// text or base64,默认text
}

var (
	addr = flag.String("addr", "", "Gitlab repository address.")	// Gitlab地址
	accessToken = flag.String("token", "", "Gitlab personal access tokens.")	// 管理员访问令牌
	secretToken = "power123"	// 认证token
)

func main()  {
	flag.Parse()

	http.HandleFunc("/", home)
	http.HandleFunc("/hooks", hooks)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func home(w http.ResponseWriter, r *http.Request)  {
	if r.URL.Path != "/" {
		w.WriteHeader(404)
		w.Write([]byte("Not Found\n"))
	} else {
		w.Write([]byte("Hello World\n"))
	}
}

func hooks(w http.ResponseWriter, r *http.Request)  {
	if r.Method == "POST" && r.Header.Get("X-Gitlab-Token") == secretToken  {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		matched, err := regexp.Match(".*project_create.*", body)
		if err != nil {
			panic(err)
		}
		if matched {
			fmt.Println(string(body))
			var project Project
			project.ID = gjson.Get(string(body), "project_id").String()
			project.Name = gjson.Get(string(body), "name").String()
			project.Path = gjson.Get(string(body), "path").String()
			project.PathWithNamespace = gjson.Get(string(body), "path_with_namespace").String()
			err := project.GetDescription()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("INFO:", project)

			if project.Language != "" && project.Env != nil {
				actions, err := project.LoadTemplate()
				if err != nil {
					panic(err)
				}

				res, err := project.CreateFile("master", actions)
				if err != nil {
					panic(err)
				}
				fmt.Println(res)
				
			}
		}
	} else {
		w.WriteHeader(401)
	}
	if r.Method == "GET" {
		w.Write([]byte("System Hooks\n"))
	}
}


// LoadTemplate 加载模板文件
func (p *Project) LoadTemplate() (actions []Action, err error) {
	if p.Language == "golang" {
		ciTmpl, err := template.ParseFiles("template/golang/gitlab-ci.tmpl")
		dockerTMPL, err := template.ParseFiles("template/golang/dockerfile.tmpl")
		if err != nil {
			return nil, err
		}
		var ciContent, dockerContent bytes.Buffer
		err = ciTmpl.Execute(&ciContent, p)
		err = dockerTMPL.Execute(&dockerContent, p)
		if err != nil {
			return nil, err
		}
		actions = []Action{
			{Action: "create", FilePath: ".gitlab-ci.yml", Content: string(ciContent.Bytes())},
			{Action: "create", FilePath: "Dockerfile", Content: string(dockerContent.Bytes())},
		}
	}
	if p.Language == "php" {
		ciTmpl, err := template.ParseFiles("template/php/gitlab-ci.tmpl")
		if err != nil {
			return nil, err
		}
		var ciContent bytes.Buffer
		err = ciTmpl.Execute(&ciContent, p)
		if err != nil {
			return nil, err
		}
		nginx, err := ioutil.ReadFile("template/php/nginx.dockerfile.tmpl")
		php, err := ioutil.ReadFile("template/php/php.dockerfile.tmpl")
		stack, err := ioutil.ReadFile("template/php/stack.tmpl")
		if err != nil {
			return nil, err
		}
		actions = []Action{
			{Action: "create", FilePath: ".gitlab-ci.yml", Content: string(ciContent.Bytes())},
			{Action: "create", FilePath: ".nginx.Dockerfile", Content: string(nginx)},
			{Action: "create", FilePath: ".php.Dockerfile", Content: string(php)},
			{Action: "create", FilePath: "stack.yaml", Content: string(stack)},
		}
	}
	if p.Language == "javascript" {
		ciTmpl, err := template.ParseFiles("template/javascript/gitlab-ci.tmpl")
		if err != nil {
			return nil, err
		}
		var ciContent bytes.Buffer
		err = ciTmpl.Execute(&ciContent, p)
		if err != nil {
			return nil, err
		}
		dockerfile, err := ioutil.ReadFile("template/javascript/dockerfile.tmpl")
		stack, err := ioutil.ReadFile("template/javascript/stack.tmpl")
		if err != nil {
			return nil, err
		}
		actions = []Action{
			{Action: "create", FilePath: ".gitlab-ci.yml", Content: string(ciContent.Bytes())},
			{Action: "create", FilePath: "Dockerfile", Content: string(dockerfile)},
			{Action: "create", FilePath: "stack.yaml", Content: string(stack)},
		}
	}
	return
}

// GetDescription 获取项目描述, 第一行为项目录描述，第二行以分号分隔，例: golang;dev,test,stable
func (p *Project) GetDescription() error {
	client := http.Client{}

	url := fmt.Sprintf("%v/api/v4/projects/%v", *addr, p.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// token: o2AD98D9xzmpQy3zqydA
	req.Header.Add("Private-Token", *accessToken)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// 解析获取到的描述信息
	p.Language, p.Env, err = parseDescription(gjson.Get(string(body), "description").String())
	if err != nil {
		return err
	}
	return nil 
}

// 解析gitlab description
func parseDescription(s string) (language string, env []string, err error) {
	match, err := regexp.Match("php|golang|javascript", []byte(s))
	if err != nil {
		return
	}
	if match {
		d := strings.Split(s, "\n")
		if len(d) < 1 {
			return
		}
		if len(d) == 1 {
			l := strings.Split(s, ";")
			language = l[0]
			env = strings.Split(l[1], ",")
		}
		if len(d) > 1 {
			l := strings.Split(d[1], ";")
			language = l[0]
			env = strings.Split(l[1], ",")
		}
	}
	return
}

// CreateFile 创建文件
func (p *Project) CreateFile(branch string, actions []Action) (response string, err error) {
	client := http.Client{}

	url := fmt.Sprintf("%v/api/v4/projects/%v/repository/commits", *addr, p.ID,)
	cf := &CreateFileOptions{
		Branch: branch,
		Actions: actions,
		CommitMessage: "增加CI/CD",
	}
	reqBody, err := json.Marshal(cf)
	if err != nil {
		return
	}
	
	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return
	}
	// token: o2AD98D9xzmpQy3zqydA
	req.Header.Add("Private-Token", *accessToken)
	req.Header.Add("Content-Type", "application/json")
	

	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	return string(body), nil
}