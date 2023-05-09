package lib 

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
	"strings"
)

type SnapshotConfigure struct {
	Prefix 			string `yaml:"prefix"`
	Suffix 			string `yaml:"suffix"`
	Target			[]SnapshotTarget `yaml:target"`
}

type SnapshotTarget struct {
	Region			string				`yaml:"region"`
	DBList 			[]ClusterTarget 	`yaml:"dblist"`
}

type ClusterTarget struct {
	DBname 			string	`yaml:"dbname"`
	Type 			string	`yaml:"type"`
	SnapshotName 	string
	Status			string
}

func CreateRDSObj(regionCode string) *rds.RDS {
	awsSess := session.Must(session.NewSession())
	// Set Regions
	rdsObj := rds.New(awsSess, aws.NewConfig().WithRegion(regionCode))

	return rdsObj
}

func ExecSnaps(wg *sync.WaitGroup,r string, k ClusterTarget) {
	defer wg.Done()
	rdsObj := CreateRDSObj(r)

	startTime := time.Now().Format("2006-01-02 15:04:05")

	checkInterval := 60*time.Second

	switch strings.ToUpper(k.Type) {
	case "CLUSTER":
		_, err := rdsObj.CreateDBClusterSnapshot(
			&rds.CreateDBClusterSnapshotInput{
				DBClusterIdentifier: aws.String(k.DBname),
				DBClusterSnapshotIdentifier: aws.String(k.SnapshotName),
			},
		)
		if err != nil {
			k.Status = "ERR"
			log.Errorf("%s",err)
			break
		}

		for {
			time.Sleep(checkInterval)

			resOut, _ := rdsObj.DescribeDBClusterSnapshots(
				&rds.DescribeDBClusterSnapshotsInput{
					DBClusterIdentifier: aws.String(k.DBname),
					DBClusterSnapshotIdentifier: aws.String(k.SnapshotName),
					SnapshotType: aws.String("manual"),
				},
			)
			if len(resOut.DBClusterSnapshots) == 0 {
				break
			}
			resStatus := *resOut.DBClusterSnapshots[0].Status

			log.Infof("Server: %s  Status: %s",k.DBname,resStatus)

			if resStatus == "creating" || resStatus == "copying" ||  resStatus == "available" {
				if resStatus == "available" {
					k.Status = "OK"
					break
				} 
				continue
			} else {
				k.Status = resStatus
				break
			}
		}
		
	case "INSTANCE":
		_, err := rdsObj.CreateDBSnapshot(
			&rds.CreateDBSnapshotInput{
				DBInstanceIdentifier: aws.String(k.DBname),
				DBSnapshotIdentifier: aws.String(k.SnapshotName),
			},
		)
		if err != nil {
			k.Status = "ERR"
			log.Errorf("%s",err)
			break
		}

		for {
			time.Sleep(checkInterval)

			resOut, _ := rdsObj.DescribeDBSnapshots(
				&rds.DescribeDBSnapshotsInput{
					DBInstanceIdentifier: aws.String(k.DBname),
					DBSnapshotIdentifier: aws.String(k.SnapshotName),
					SnapshotType: aws.String("manual"),
				},
			)
			
			resStatus := *resOut.DBSnapshots[0].Status

			log.Infof("Server: %s  Status: %s",k.DBname,resStatus)

			if resStatus == "creating" || resStatus == "copying" ||  resStatus == "available" {
				if resStatus == "available" {
					k.Status = "OK"
					break
				} 
				continue
			} else {
				k.Status = resStatus
				break
			}
		}
	}

	endTime := time.Now().Format("2006-01-02 15:04:05")
	log.Infof("[%s]Snpashots : %s Start Time : %s EndTime : %s",k.Status,k.SnapshotName,startTime,endTime)
	fmt.Println(fmt.Sprintf("[%s]Snpashots : %s Start Time : %s EndTime : %s",k.Status,k.SnapshotName,startTime,endTime))
}

func (s ClusterTarget) SnapshotSet(pf string, sf string) string {
	snaps := fmt.Sprintf("%s-%s-%s",pf,s.DBname,sf)
	return snaps
}

func GetClusterStatus(r string,c string) (string, error) {
	rdsObj := CreateRDSObj(r)

	// Describe Clusters
	descCluster, err := rdsObj.DescribeDBClusters(
		&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(c),
		})
	if err != nil {
		return "", err
	}

	return *descCluster.DBClusters[0].Status, nil
}

func GetInstanceStatus(r string,c string) (string, error) {
	rdsObj := CreateRDSObj(r)

	// Describe Clusters
	descInstance, err := rdsObj.DescribeDBInstances(
		&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(c),
		})
	if err != nil {
		return "", err
	}

	return *descInstance.DBInstances[0].DBInstanceStatus, nil
}

