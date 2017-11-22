/*
Author: Sriram Kaushik
Master Node that issues requests to slave (master node is more of a client here. Slave nodes are server nodes)
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kaushiksriram100/deploy-splunk-uf-dist/shyunutils"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

const maxretries int = 5 //max number of times to retry with different slaves. helps in calling the recursion function dialTCP
var count int = 0        //count the number of attempts made.

func DialTCP(requests []shyunutils.RequestMessage, slavenodes *string, ansible_playbook_path *string, ansible_playbook_action *string, oneops_jar_path *string, targettype *string, logfile *os.File) {
	log.SetOutput(logfile)
	//fmt.Println("-------------", requests)
	//make the slave nodes as a array.
	slavenode := strings.Split((*slavenodes), ",")

	//for each request, we will pick a random remote server and use dialTCP to send the request.
	//if there is some connection issue, we will recursively call this DialTCP to try with some other host.
	for _, v := range requests {

		v.PlaybookAction = (*ansible_playbook_action)
		v.PlaybookPath = (*ansible_playbook_path)
		v.InvJar = (*oneops_jar_path)

		//pick a random number to send to a random remote server
		trynode := 0
		if len(slavenode) > 1 {
			rand.Seed(time.Now().UTC().UnixNano())
			trynode = rand.Intn(len(slavenode))
		}

	//	fmt.Println("trying to resolve IP")

		//Try to resolve, if fails, move to another node.
		tcpAddr, err := net.ResolveTCPAddr("tcp", slavenode[trynode])

		//fmt.Println("resolve Error?", err)

		if err != nil {
			if count < maxretries {
				var tmp []shyunutils.RequestMessage
				tmp = append(tmp, v)
				log.Print("WARNING: Slave node not reachable. Will attempt some other host - " + slavenode[trynode])
				count = count + 1
				DialTCP(tmp, slavenodes, ansible_playbook_path, ansible_playbook_action, oneops_jar_path, targettype, logfile)
				continue
			} else {
				continue
			}

		}

	//	fmt.Println("dialing tcp")
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
	//	fmt.Println("dial error", err)

		if err != nil {
			if count < maxretries {
				var tmp1 []shyunutils.RequestMessage
				tmp1 = append(tmp1, v)
				log.Print("ERROR:Not able to connect to remote host, will try with some other host - ", tcpAddr, tmp1)
				count = count + 1
				DialTCP(tmp1, slavenodes, ansible_playbook_path, ansible_playbook_action, oneops_jar_path, targettype, logfile)
				continue

			} else {
				continue
			}

		}

		sendjson := json.NewEncoder(conn)

		err = sendjson.Encode(v)

		if err != nil {
			log.Print("ERROR: ERROR sending request", slavenode[trynode])

		}

		log.Print("INFO: Sent request to slave", slavenode[trynode])
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

	// set a seed for picking a random number
	rand.Seed(time.Now().UnixNano())

	//break the slice requestmessage and send it

	DialTCP(requestmessage, slavenodes, ansible_playbook_path, ansible_playbook_action, oneops_jar_path, target_type, logfile)

}
