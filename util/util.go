package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-git/go-git/v5"
	"github.com/one-meta/meta-cli/entity"
	"github.com/sethvargo/go-password/password"
	"github.com/spf13/viper"
)

type Answer struct {
	Env string
}

var (
	CFG = &entity.Config{}
	// the questions to ask
	qs = []*survey.Question{
		{
			Name: "env",
			Prompt: &survey.Select{
				Message: "请选择本地开发或者线上环境:",
				Options: []string{"dev", "prod"},
				Default: "prod",
				Help:    "\ndev:本地开发，配置文件host => 127.0.0.1【本地运行后端和前端服务】\nprod:线上环境，配置文件host => 各个容器别名【docker compose运行】",
			},
		},
	}
	Separator = string(os.PathSeparator)
)

func init() {
	LoadConfig()
}

func ReNewProject() {
	fileExist := checkFiles("meta", []string{getSeperator("resource", "config.toml"), "Docker-Compose.yaml"})
	if fileExist {
		fmt.Println("Initialization has been run, meta/data will be delete and current password will be replace if continue.")
		fmt.Println("Are you sure? \n[y]es or [n]o (default: no):")
		var check string
		fmt.Scanln(&check)
		if check != "y" {
			fmt.Println("Canceled")
			os.Exit(0)
		}
		// delete meta/data
		removeFiles("meta", []string{"data"}, true)
	}

	answers := Answer{}

	// perform the questions
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	localDev := true
	if answers.Env == "prod" {
		localDev = false
	}

	cloneAndInitFile(false, localDev)
	fmt.Println("done.")
}

func NewProject() {
	fileExist := checkFiles("meta", []string{getSeperator("resource", "config.toml"), "Docker-Compose.yaml"})
	if fileExist {
		fmt.Println("Initialization has been run, exit.")
		os.Exit(0)
	}

	answers := Answer{}

	// perform the questions
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	localDev := true
	if answers.Env == "prod" {
		localDev = false
	}

	cloneAndInitFile(true, localDev)
	fmt.Println("done.")
}

func cloneAndInitFile(firstLoad, localDev bool) {
	pwdMap := make(map[string]string)

	// schema := "http"
	// domain := "gitlab.local"
	schema := "https"
	domain := "github.com"
	baseUrl := fmt.Sprintf("%s://%s", schema, domain)
	// 检测部分依赖
	dependArrays := []string{"meta-g", "meta-front-g"}
	exitFlag := checkExecute(domain, dependArrays)
	if exitFlag {
		os.Exit(0)
	}
	// clone项目
	if firstLoad {
		// git clone 后端项目
		GitClone("meta", baseUrl)
		//////git clone 前端项目
		GitClone("meta-front", baseUrl)
		//////git clone meta-front-g => config.toml和BasePage
		GitClone("meta-front-g", baseUrl)
	}

	checkFiles("meta", []string{getSeperator(".template", "config.toml"), getSeperator(".template", "Docker-Compose.yaml")})
	checkFiles("meta-front", []string{getSeperator(".template", "Docker-Compose.yaml")})
	checkFiles("meta-front-g", []string{"config.toml", getSeperator("BasePage", "Detail.tsx"), getSeperator("BasePage", "index.tsx")})
	// init
	for _, v := range CFG.Password.Arrays {
		pwd, err := password.Generate(24, 8, 0, false, false)
		if err != nil {
			log.Fatal(err)
		}
		pwdMap[v] = pwd
	}
	initTemplate(pwdMap, localDev, "meta", getSeperator("resource", "config.toml"), getSeperator(".template", "config.toml"))
	initTemplate(pwdMap, localDev, "meta", "Docker-Compose.yaml", getSeperator(".template", "Docker-Compose.yaml"))
	initTemplate(pwdMap, localDev, "meta-front", "Docker-Compose.yaml", getSeperator(".template", "Docker-Compose.yaml"))
}

func initTemplate(pwdMap map[string]string, localDev bool, rootPath, targetFile, sourceFile string) {
	removeFiles(rootPath, []string{targetFile}, false)

	sourcePath := getSeperator(rootPath, sourceFile)
	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
	file, err := os.Open(sourcePath)
	resultFile, err := os.Create(getSeperator(rootPath, targetFile))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	defer resultFile.Close()
	line := bufio.NewReader(file)
	writer := bufio.NewWriter(resultFile)
	for {
		content, _, err := line.ReadLine()
		if err == io.EOF {
			break
		}
		target := string(content)
		if localDev {
			// 后端配置文件
			if rootPath == "meta" && strings.HasPrefix(target, "host = ") {
				target = "host = \"127.0.0.1\""
			}
		}
		// 模板单行数据中包含_password
		if strings.Contains(target, "_password") {
			for k, v := range pwdMap {
				if strings.Contains(target, k) {
					target = strings.ReplaceAll(target, k, v)
					break
				}
			}
		}

		fmt.Fprintln(writer, target)
	}
	writer.Flush()
}

func checkExecute(baseUrl string, dependArrays []string) bool {
	for _, name := range dependArrays {
		cmd := exec.Command(name)
		err := cmd.Start()
		if err != nil {
			log.Printf("%s not installed, please install with:", name)
			log.Printf("go install %s/one-meta/%s@latest", baseUrl, name)
			return true
		}
	}
	return false
}

func GitClone(targetPath, baseUrl string) {
	repository := fmt.Sprintf("%s/one-meta/%s.git", baseUrl, targetPath)
	log.Println("git clone", repository)
	GitClone2Target(targetPath, repository)
	// 删除 .git .github 文件
	removeGitFile(targetPath)
}

func removeGitFile(targetPath string) {
	// 删除 .git .github 文件
	removeFiles(targetPath, []string{".git", ".github"}, true)
}

func removeFiles(rootPath string, files []string, show bool) {
	for _, v := range files {
		targetPath := getSeperator(rootPath, v)
		if _, err := os.Stat(targetPath); err != nil {
			if os.IsExist(err) {
				if show {
					log.Println("remove", targetPath)
				}
				err := os.RemoveAll(targetPath)
				if err != nil {
					fmt.Println("remove file err ", err)
				}
			}
		}
	}
}

func checkFiles(rootPath string, files []string) bool {
	for _, v := range files {
		targetPath := getSeperator(rootPath, v)
		if _, err := os.Stat(targetPath); err != nil {
			if os.IsNotExist(err) {
				// log.Fatalf("File: %s not exist", targetPath)
				return false
			}
		}
		return true
	}
	return false
}

func GitClone2Target(targetPath, repository string) {
	_, err := git.PlainClone(targetPath, false, &git.CloneOptions{
		URL:      repository,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("config.toml not found, create one")
		viper.SetDefault("password", map[string][]string{
			"arrays": {
				"meta_password", "meta_mysql_root_password",
				"meta_mysql_password", "meta_mariadb_password",
				"meta_postgres_password", "meta_redis_password",
				"meta_jwt_password",
			},
		})
		viper.SafeWriteConfigAs("config.toml")
	}
	err = viper.Unmarshal(&CFG)
	return err
}

func getSeperator(filePath ...string) string {
	return strings.Join(filePath, Separator)
}
