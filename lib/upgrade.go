package lib 

import (
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	log "github.com/sirupsen/logrus"
	"errors"
	"strings"
	"fmt"
	"time"
	"sync"
)


type UpgradeDivInfo struct {
	Engine 			string 	`yaml:"engine"`
	Region 			string 	`yaml:"region"`
	From  			string 	`yaml:"from"`
	To 				string	`yaml:"to"`
	DBlist 			[]ClusterTarget 	`yaml:"dblist"`
}

func GetActiveVersion(e string) ([]string,error) {
	var eList []string
	rdsObj := CreateRDSObj("ap-northeast-2")

	descVersion, err := rdsObj.DescribeDBEngineVersions(
		&rds.DescribeDBEngineVersionsInput{
			Engine : aws.String(e),
		},
	)
	if err != nil {
		return eList,err
	}
	
	if len(descVersion.DBEngineVersions) == 0 {
		return eList, errors.New("Check your AWS token.")
	}

	for _, v := range descVersion.DBEngineVersions {
		eList = append(eList,*v.EngineVersion)
	}

	return eList, nil
}

func CheckUpgradeEngine(r string, ce string, c ClusterTarget,fe string, te string) (bool, error) {
	availableUpgrade := false
	rdsObj := CreateRDSObj(r)
	
	switch strings.ToUpper(c.Type) {
	case "CLUSTER":
		data, err := GetCluster(rdsObj,c.DBname)
		if err != nil {
			return availableUpgrade, err
		}

		if ce == *data.Engine {
			if fe == *data.EngineVersion {
				availableUpgrade = true
				return availableUpgrade, nil
			} else {
				return availableUpgrade, errors.New("Not Matched Engine Version.")
			}
		} else {
			return availableUpgrade, errors.New("Not Matched Engine.")
		}
		
	case "INSTANCE":
		data, err := GetInstance(rdsObj,c.DBname)
		if err != nil {
			return availableUpgrade, err
		}

		if ce == *data.Engine {
			if fe == *data.EngineVersion {
				availableUpgrade = true
				return availableUpgrade, nil
			} else {
				return availableUpgrade, errors.New("Not Matched Engine Version.")
			}
		} else {
			return availableUpgrade, errors.New("Not Matched Engine.")
		}
	}
	return availableUpgrade, nil
}

func GetCluster(rdsObj *rds.RDS,c string) (*rds.DBCluster, error) {
	// Describe Clusters
	descCluster, err := rdsObj.DescribeDBClusters(
		&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(c),
		})
	if err != nil {
		return nil, err
	}
	
	return descCluster.DBClusters[0], nil
}

func GetInstance(rdsObj *rds.RDS,c string) (*rds.DBInstance, error) {
	// Describe Clusters
	descInstance, err := rdsObj.DescribeDBInstances(
		&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(c),
		})
	if err != nil {
		return nil, err
	}

	return descInstance.DBInstances[0], nil
}

func ModifyEngineUpgrade(wg *sync.WaitGroup, r string,k ClusterTarget, te string) {
	defer wg.Done()
	rdsObj := CreateRDSObj(r)

	snapStartTime := time.Now().Format("2006-01-02 15:04:05")

	snapInterval := 60*time.Second

	snapSuffix := time.Now().Format("20060102")
	k.SnapshotName = fmt.Sprintf("upgrade-%s-%s",k.DBname,snapSuffix)

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
			time.Sleep(snapInterval)

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
			time.Sleep(snapInterval)

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

	snapEndTime := time.Now().Format("2006-01-02 15:04:05")
	log.Infof("[%s]Snpashots : %s Start Time : %s EndTime : %s",k.Status,k.SnapshotName,snapStartTime,snapEndTime)
	fmt.Println(fmt.Sprintf("[%s]Snpashots : %s Start Time : %s EndTime : %s",k.Status,k.SnapshotName,snapStartTime,snapEndTime))


	// Upgrade Part
	startTime := time.Now().Format("2006-01-02 15:04:05")

	checkInterval := 600*time.Second

	switch strings.ToUpper(k.Type) {
	case "CLUSTER":
		_, err := rdsObj.ModifyDBCluster(
			&rds.ModifyDBClusterInput{
				DBClusterIdentifier : aws.String(k.DBname),
				EngineVersion: aws.String(te),
				ApplyImmediately: aws.Bool(true),
			},
		)
		
		if err != nil {
			k.Status = "ERR"
			log.Errorf("%s",err)
			break
		}

		for {
			time.Sleep(checkInterval)

			resOut, _ := rdsObj.DescribeDBClusters(
				&rds.DescribeDBClustersInput{
					DBClusterIdentifier: aws.String(k.DBname),
				},
			)

			if len(resOut.DBClusters) == 0 {
				break
			}

			resStatus := *resOut.DBClusters[0].Status
			log.Infof("Server: %s  Status: %s",k.DBname,resStatus)

			if resStatus == "upgrading" || resStatus == "available" {
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
		_, err := rdsObj.ModifyDBInstance(
			&rds.ModifyDBInstanceInput{
				DBInstanceIdentifier : aws.String(k.DBname),
				EngineVersion: aws.String(te),
				ApplyImmediately: aws.Bool(true),
			},
		)
		
		if err != nil {
			k.Status = "ERR"
			log.Errorf("%s",err)
			break
		}

		for {
			time.Sleep(checkInterval)

			resOut, _ := rdsObj.DescribeDBInstances(
				&rds.DescribeDBInstancesInput{
					DBInstanceIdentifier: aws.String(k.DBname),
				},
			)

			if len(resOut.DBInstances) == 0 {
				break
			}

			resStatus := *resOut.DBInstances[0].DBInstanceStatus
			log.Infof("Server: %s  Status: %s",k.DBname,resStatus)

			if resStatus == "upgrading" || resStatus == "available" {
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
	log.Infof("[%s]DB : %s Engine : %s Start Time : %s EndTime : %s",k.Status,k.DBname,te,startTime,endTime)
	fmt.Println(fmt.Sprintf("[%s]DB : %s Engine : %s Start Time : %s EndTime : %s",k.Status,k.DBname,te,startTime,endTime))

}