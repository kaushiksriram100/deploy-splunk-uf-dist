package shyunutils

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type RequestMessage struct {
	Nspath_file string `json:"nspath"`
	//OO_API_Token   string `json:"oo_api_token"`
	OO_Org         string `json:"oo_org"`
	OO_Assembly    string `json:"oo_assembly"`
	OO_Platform    string `json:"oo_platform"`
	OO_Env         string `json:"oo_env"`
	PlaybookAction string `json:"playbookaction"`
	PlaybookPath   string `json:"playbookpath"`
	InvJar         string `json:"invjar"`
}

var request []RequestMessage

func CreateLogFile(log_file_path *string, logfilename string) (*os.File, error) {

	//Make sure the directory path exists, if not create it.

	var logfile *os.File

	if err := os.MkdirAll((*log_file_path), 0744); err != nil {
		return logfile, errors.New("ERROR:Failed to create log file path. Check user permissions")
	}

	//create the actual log file.

	logfile, err := os.OpenFile((*log_file_path)+"/"+logfilename+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//we will not close the logfile here. We can close it in the main function. logfile is *os.File (it's an address so it's ok to close in main before exiting)

	if err != nil {
		return logfile, errors.New("ERROR: Can't create log file. I will not start.. sorry")
	}

	return logfile, nil

}

func PopulateHash(path string, fp os.FileInfo, err error) error {
	if fp.IsDir() {
		return nil
	}

	if filename := fp.Name(); filename != "settings.yml" {
		return nil
	}

	if path == "" {
		return nil
	}

	tmp := strings.Split(path, "/")

	var tmprequest RequestMessage

	tmprequest.OO_Env = tmp[len(tmp)-2]
	tmprequest.OO_Platform = tmp[len(tmp)-3]
	tmprequest.OO_Assembly = tmp[len(tmp)-4]
	tmprequest.OO_Org = tmp[len(tmp)-5]
	tmprequest.Nspath_file = tmprequest.OO_Org + "_" + tmprequest.OO_Assembly + "_" + tmprequest.OO_Platform + "_" + tmprequest.OO_Env

	request = append(request, tmprequest)

	return nil
}

func ExtractEnvVars(target_type, ansible_playbook_path *string, logfile *os.File) ([]RequestMessage, error) {

	log.SetOutput(logfile)

	//we will extract env variables from the path of the files itself. we will use filewalk.

	//add a validator to check root_dir variable (the path) if files exists underneath else return here. If path doesn't exist then we will get a nil pointer exception.

	root_dir := (*ansible_playbook_path) + "/vars/conf/" + (*target_type)

	request = make([]RequestMessage, 0, 3)

	err := filepath.Walk(root_dir, PopulateHash)

	if err != nil {
		log.Fatal("ERROR: Could not map assembly/platform/env together. Check the hierarchy")
		return request, errors.New("error")
	}

	//I have to return map[int]EnvVars because I can't pass a pointer to PopulateHash function. This will have some perf issues.

	return request, nil

}
