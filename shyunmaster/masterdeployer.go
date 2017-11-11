//Master Node that issues requests to slave (master node is more of a client here. Slave nodes are server nodes)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kaushiksriram100/deploy-splunk-uf-dist/shyunutils"
	"log"
	"net"
	"os"
	"strings"
)

func DialTCP(requests []shyunutils.RequestMessage, slavenodes *string, ansible_playbook_path *string, ansible_playbook_action *string, oneops_jar_path *string, targettype *string, logfile *os.File) {
	log.SetOutput(logfile)
	slavenode := strings.Split((*slavenodes), ",")

	tcpAddr, err := net.ResolveTCPAddr("tcp", slavenode[0])

	if err != nil {
		fmt.Println("ERROR: slavenode not reachable")
		return
	}

	for _, v := range requests {

		v.PlaybookAction = (*ansible_playbook_action)
		v.PlaybookPath = (*ansible_playbook_path)
		v.InvJar = (*oneops_jar_path)

		conn, err := net.DialTCP("tcp", nil, tcpAddr)

		if err != nil {
			log.Print("ERROR:Not able to connect to remote host", tcpAddr)
			return
		}

		sendjson := json.NewEncoder(conn)

		err = sendjson.Encode(v)
		fmt.Println(v)
		fmt.Printf("%T", v)

		if err != nil {
			log.Print("ERROR: ERROR sending request")
		}

		log.Print("INFO: Sent request to slave")
		conn.Close()

	}

}

func main() {

	// this main fuction for sending tasks to slave through dialTCP

	slavenodes := flag.String("slaves", "localhost:3001", "all slave nodes, comma separated")
	//Get logfile path and config file path from arguments.
	var log_file_path = flag.String("logfile", "/var/tmp/deploy-splunk-uf/log/", "--logfile=logfile path. Default is /var/tmp/deploy-splunk-uf/log/")

	//Get the ansible playbook file and oneops-inventory jar path.

	var ansible_playbook_path = flag.String("playbookpath", "/Users/skaush1/Documents/my_dev_env/ansible-workspace/wm-splunk-universal-forwarder/", "--playbookpath <fullpath of playbook>")
	var ansible_playbook_action = flag.String("playbookaction", "main.yml", "--playbookaction <main.yml, start.yml, stop.yml>. default main.yml")
	var oneops_jar_path = flag.String("invjar", "/Users/skaush1/Documents/my_dev_env/oneops-inventory/oo-wrapper.py", "--invjar <fullpath of oneops inv jar. check documentation>")
	var target_type = flag.String("targettype", "oneops", "--targettype oneops or physical. This the first directory in conf hierarchy")

	flag.Parse()

	//create a log file for master

	logfile, err := shyunutils.CreateLogFile(log_file_path, "shyunmaster")

	if err != nil {
		fmt.Println("ERROR: Can't open log file. ")
		return
	}

	//need to set the output
	log.SetOutput(logfile)

	//This will avoid any memory leaks when the program ends.
	defer logfile.Close()

	//At this points we have the log files in place. Now let's start some heavy lifting. We need to form a slice of RequestMessages

	requestmessage, err := shyunutils.ExtractEnvVars(target_type, ansible_playbook_path, logfile) //requestmessage is []RequestMessage

	if err != nil {
		log.Print("ERROR: Cant extract vars")
		return
	}

	//Now requestmessages has everything in each slice.

	//send json data.

	//break the slice requestmessage and send it

	DialTCP(requestmessage, slavenodes, ansible_playbook_path, ansible_playbook_action, oneops_jar_path, target_type, logfile)

}
