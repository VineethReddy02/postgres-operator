/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	PostgresqlLister "github.com/zalando/postgres-operator/pkg/generated/clientset/versioned/typed/acid.zalan.do/v1"
	"strconv"
	"k8s.io/client-go/rest"
)

// connectCmd represents the kubectl pg connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connects to the shell prompt, psql prompt of postgres cluster",
	Long: `Connects to the shell prompt, psql prompt of postgres cluster and also to specified replica or master.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterName,_ := cmd.Flags().GetString("clusterName")
		master,_ := cmd.Flags().GetBool("master")
		replica,_ := cmd.Flags().GetString("replica")
		psql,_ := cmd.Flags().GetBool("psql")
		user,_ := cmd.Flags().GetString("user")
		if psql {
			if user == "" {
				log.Fatal("please provide user for psql prompt")
			}
		}
		connect(clusterName,master,replica,psql,user)
	},
}

func connect(clusterName string,master bool,replica string,psql bool,user string) {
	config := getConfig()
	client,er := kubernetes.NewForConfig(config)
	if er != nil {
		log.Fatal(er)
	}
	postgresConfig, err := PostgresqlLister.NewForConfig(config)
	postgresCluster, err := postgresConfig.Postgresqls(getCurrentNamespace()).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	numOfInstances := postgresCluster.Spec.NumberOfInstances
	var podName string
	var podRole string
	for ins:=0;ins < int(numOfInstances);ins++ {
		pod,err := client.CoreV1().Pods(getCurrentNamespace()).Get(clusterName+"-"+strconv.Itoa(ins),metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		podRole = pod.Labels["spilo-role"]
		if podRole == "master" && master {
			podName = pod.Name
			fmt.Printf("connected to %s with name %s\n",podRole, podName)
			break
		} else if podRole == "replica" &&  !master  && (pod.Name == replica || replica == "") {
			podName = pod.Name
			fmt.Printf("connected to %s with pod name as %s\n",podRole, podName)
			break
		}
	}
	execRequest := &rest.Request{}
	if psql {
		execRequest = client.CoreV1().RESTClient().Post().Resource("pods").
			Name(podName).
			Namespace("default").
			SubResource("exec").
			Param("container", "postgres").
			Param("command", "psql").
			Param("command", "-U").
			Param("command", user).
			Param("stdin", "true").
			Param("stdout", "true").
			Param("stderr", "true").
			Param("tty", "true")
	} else {
		execRequest = client.CoreV1().RESTClient().Post().Resource("pods").
			Name(podName).
			Namespace("default").
			SubResource("exec").
			Param("container", "postgres").
			Param("command", "su").
			Param("command", "postgres").
			Param("stdin", "true").
			Param("stdout", "true").
			Param("stderr", "true").
			Param("tty", "true")
	}
	exec,err := remotecommand.NewSPDYExecutor(config,"POST",execRequest.URL())
	if err != nil {
		log.Fatal(err)
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin: os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty: true,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	connectCmd.Flags().StringP("clusterName", "c", "", "provide the cluster name.")
	connectCmd.Flags().BoolP("master", "m", false, "connect to master.")
	connectCmd.Flags().StringP("replica", "r","", "connect to replica.")
	connectCmd.Flags().BoolP("psql","p",false,"connect to psql prompt.")
	connectCmd.Flags().StringP("user","u","","provide user.")
	rootCmd.AddCommand(connectCmd)
}
