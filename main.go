package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type IPGroupController struct {
	targetGroupARN string
}

var region = flag.String("region", "", "AWS Region")
var targetARN = flag.String("targetgrouparn", "", "ARN of the Target group to update")
var service = flag.String("service", "", "Name of the service to watch")
var port = flag.Int64("port", 0, "Port number")
var httpAddr = flag.String("http", ":8080", "Address to bind for metrics server")

func main() {
	flag.Parse()

	if *region == "" || *targetARN == "" || *service == "" || *port == 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

	// Start metrics server
	http.Handle("/metrics", promhttp.Handler())
	go func() { log.Fatal(http.ListenAndServe(*httpAddr, nil)) }()

	// Setup AWS
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	elbSvc := elbv2.New(sess, &aws.Config{Region: region})
	log.Println("Port is custom")

	metadata := ec2metadata.New(sess)
	currentVpc, err := getVpcIDFromEC2Metadata(metadata)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Current vpc is %s\n", currentVpc)
	targetVpc, err := getTargetGroupVpcID(elbSvc, targetARN)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("targetVpc vpc is %s\n", targetVpc)
	isOtherVpc := currentVpc != targetVpc
	tg := TargetGroup{
		ARN:        targetARN,
		connection: elbSvc,
		Port:       *port,
	}
	tg.UpdateKnown()

	watcher := Watcher{Service: *service, port: *port, tg: &tg, isOtherVpc: isOtherVpc}

	// Setup k8s
	client, ns := NewK8sClient()
	watchOptions := meta.ListOptions{}

	for {
		watch, err := client.CoreV1().Endpoints(ns).Watch(watchOptions)
		if err != nil {
			log.Fatal("Opening watch failed,", err)
		}
		watchChan := watch.ResultChan()
		watcher.Watch(watchChan)
	}
}

func getTargetGroupVpcID(elbSvc *elbv2.ELBV2, targetARN *string) (string, error) {
	tgs := []*string{targetARN}
	input := elbv2.DescribeTargetGroupsInput{TargetGroupArns: tgs}
	out, err := elbSvc.DescribeTargetGroups(&input)
	if err != nil {
		return "", err
	}
	return *out.TargetGroups[0].VpcId, err
}

func getVpcIDFromEC2Metadata(metadata *ec2metadata.EC2Metadata) (string, error) {
	mac, err := metadata.GetMetadata("mac")
	if err != nil {
		return "", err
	}
	vpcID, err := metadata.GetMetadata(fmt.Sprintf("network/interfaces/macs/%s/vpc-id", mac))
	if err != nil {
		return "", err
	}
	return vpcID, nil
}
