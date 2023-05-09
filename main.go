package main

import (
	log "github.com/sirupsen/logrus"
	"fmt"
	"strings"
	"strconv"
	"os"
	"sync"
	"kdba-pm-toolkit/lib"
	"gopkg.in/yaml.v3"
	"flag"
	"io/ioutil"
	"golang.org/x/crypto/ssh/terminal"
	"time"
	"errors"
)

const (
	appName	= "rds-toolkit"
	desc 	= "rds-toolkit by DBA Maintenance"
	version	= "v0.1.0"
)

func main() {
	var configPath,logPath string
	flag.StringVar(&configPath,"conf","./toolkit-setup.yml","Toolkit Setup Yaml")	
	flag.StringVar(&logPath,"log","./toolkit.log","kdba Toolkit Log")
	flag.Parse()

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	} else {
		log.SetOutput(f)
	}

	Config, err := ConfLoader(configPath)
	if err != nil {
		fmt.Println("Config Load failed.",err)
		log.Panic(err)
	}

	optList := []string{
		"Configure Reading(test)", // 0
		"Create RDS Snapshot",
		"Describe Active DB Versions",
		"DB Engine Upgrade",
		"DB Connect Test",
		"DB Cache Warming",
	}

	fmt.Println(" ")
	fmt.Println(appName,version)
	fmt.Println(desc)
	fmt.Println("\nOptions:")
	for i, v := range optList {
		fmt.Println(fmt.Sprintf("%d) %s",i,v))
	}

	optStr, err := lib.GetOpt("Select Option")
	if err != nil {
		lib.Exiter(err)
	}

	opt, _ := strconv.Atoi(*optStr)

	switch opt {
	case 0:
		fmt.Println("Configure Reading:")
		fmt.Println(Config)
	case 1: // RDS Backup
		snapConf := Config.Snapshot
		
		fmt.Println(fmt.Sprintf("Run %s. ",optList[opt]))

		fmt.Println("Current Configure:")
		fmt.Println("===========================================================================")
		fmt.Println(fmt.Sprintf("Prefix : %s / Suffix : %s",snapConf.Prefix,snapConf.Suffix))

		var dbCount int
		for i, v := range snapConf.Target {
			for si, sv := range v.DBList {
				// DB Type Check
				switch strings.ToUpper(sv.Type){
				case "CLUSTER":
					status , err := lib.GetClusterStatus(v.Region,sv.DBname)
					if err != nil {
						fmt.Println(err)
						log.Errorf("%s",err)
					}

					lib.Print(fmt.Sprintf("[CHECK]%s(%s) - %s",sv.DBname,v.Region,status))
				case "INSTANCE":
					status , err := lib.GetInstanceStatus(v.Region,sv.DBname)
					if err != nil {
						fmt.Println(err)
						log.Errorf("%s",err)
					}

					lib.Print(fmt.Sprintf("[CHECK]%s(%s) - %s",sv.DBname,v.Region,status))
				}

				// Snapshot Naming and init
				dbCount++
				snapConf.Target[i].DBList[si].SnapshotName = sv.SnapshotSet(snapConf.Prefix,snapConf.Suffix)
				snapConf.Target[i].DBList[si].Status = "Unavailable"
	
				lib.Print(fmt.Sprintf("DB : %s(%s) / Snapshot : %s",snapConf.Target[i].DBList[si].DBname,v.Region,snapConf.Target[i].DBList[si].SnapshotName))
			}
		}
		fmt.Println("")
	
		continueDiv, err := lib.GetOpt("Continue(Y/N)")
		if err != nil {
			lib.Exiter(err)
		}

		if strings.ToUpper(*continueDiv) != "Y" {
			break
		}

		var wg sync.WaitGroup
		
		wg.Add(dbCount)
		fmt.Println("")
		fmt.Println("Processing:")
		fmt.Println("===========================================================================")
		for i, v := range snapConf.Target {
			for si, sv := range v.DBList {
				lib.Print(fmt.Sprintf("[REQ]Create Snapshot %s for %s.",snapConf.Target[i].DBList[si].SnapshotName,snapConf.Target[i].DBList[si].DBname))
				go lib.ExecSnaps(
					&wg,
					v.Region,
					sv,
				)
			}
			
		}

		wg.Wait()

		lib.Print("Process Complete.")
	case 2: // Active Engine Version

		fmt.Println(fmt.Sprintf("Run %s. ",optList[opt]))

		engineList := []string{
			"aurora-mysql",
			"aurora-postgresql",
			"mysql",
			"postgres",
		}
		fmt.Println("\nEngine:")
		for i, v := range engineList {
			fmt.Println(fmt.Sprintf("%d) %s",i,v))
		}

		engineStr, err := lib.GetOpt("Select Engine")
		if err != nil {
			lib.Exiter(err)
		}

		engineNo, _ := strconv.Atoi(*engineStr)
		if engineNo > len(engineList)-1 {
			fmt.Println("Check Engine No.")
			break
		}

		engineVersionList, err := lib.GetActiveVersion(engineList[engineNo])
		if err != nil {
			lib.Exiter(err)
		}
		fmt.Println("\nEngine Versions:")
		for _, v := range engineVersionList {
			fmt.Println(v)
		}
	case 3: // Engine Upgrade
		fmt.Println(fmt.Sprintf("Run %s. ",optList[opt]))

		upgradeConf := Config.Upgrade

		fmt.Println("Version Check:")
		fmt.Println("===========================================================================")

		exitSignal := 0
		var dbCount int
		for _, v := range upgradeConf {
			fmt.Println("")
			lib.Print(fmt.Sprintf("Engine : %s / Version : %s -> %s",v.Engine, v.From,v.To))
			for _, d := range v.DBlist {
				upgradeCheck, err := lib.CheckUpgradeEngine(v.Region,v.Engine,d,v.From,v.To)
				lib.Print(fmt.Sprintf("[Check]%s - %v",d.DBname,upgradeCheck))
				if err != nil {
					fmt.Println(err)
					log.Errorf("%s",err)
				}

				if upgradeCheck == false {
					exitSignal = 1
				}

				dbCount++
			}
		}

		fmt.Println("")
		if exitSignal == 1 { 
			lib.Exiter(errors.New("Check Upgrade Configure."))
		}
	
		continueDiv, err := lib.GetOpt("Continue(Y/N)")
		if err != nil {
			lib.Exiter(err)
		}

		if strings.ToUpper(*continueDiv) != "Y" {
			break
		}

		var wg sync.WaitGroup
		fmt.Println("*Snapshots are taken first.")
		wg.Add(dbCount)
		fmt.Println("")
		fmt.Println("Processing:")
		fmt.Println("===========================================================================")

		for _, v := range upgradeConf {
			for _,d := range v.DBlist {
				lib.Print(fmt.Sprintf("[REQ]Upgrade Engine From %s to %s for %s",v.From,v.To,d.DBname))
				go lib.ModifyEngineUpgrade(
					&wg,
					v.Region,
					d,
					v.To,
				)
			}
		}

		wg.Wait()

		lib.Print("Process Complete.")


	case 4: // DB Connect Test

		fmt.Println(fmt.Sprintf("Run %s. ",optList[opt]))

		engineList := []string{
			"mysql",
			"postgres",
		}
		fmt.Println("\nEngine:")
		for i, v := range engineList {
			fmt.Println(fmt.Sprintf("%d) %s",i,v))
		}

		engineStr, err := lib.GetOpt("Select Engine")
		if err != nil {
			lib.Exiter(err)
		}

		engineNo, _ := strconv.Atoi(*engineStr)
		if engineNo > len(engineList)-1 {
			fmt.Println("Check Engine No.")
			break
		}

		endPoint, err := lib.GetOpt("Endpoint")
		if err != nil {
			lib.Exiter(err)
		}

		portStr, err := lib.GetOpt("Port")
		if err != nil {
			lib.Exiter(err)
		}

		port, _ := strconv.Atoi(*portStr)

		dbUsers, err := lib.GetOpt("User")
		if err != nil {
			lib.Exiter(err)
		}

		fmt.Printf("Password : ")
		vPass, err := terminal.ReadPassword(0)
		if err != nil {
			lib.Exiter(err)
		}
		var dbPass string 
		dbPass = string(vPass)
		fmt.Println("")


		db, err := lib.GetOpt("Database")
		if err != nil {
			lib.Exiter(err)
		}

		switch engineNo {
		case 0: // mysql
			dbObj, err := lib.GetDBObject(*endPoint,port,*dbUsers,dbPass,*db)
			if err != nil {
				lib.Exiter(err)
			}
			defer dbObj.Close()
			lib.Print(fmt.Sprintf("[OK]%s Connect for %s.",*dbUsers,*endPoint))
			fmt.Println("")
		case 1: // postgresql
			dbObj, err := lib.GetPostObject(*endPoint,port,*dbUsers,dbPass,*db)
			if err != nil {
				lib.Exiter(err)
			}
			defer dbObj.Close()
			lib.Print(fmt.Sprintf("[OK]%s Connect for %s.",*dbUsers,*endPoint))
			fmt.Println("")
		}

	case 5: // DB Cache Warming
		fmt.Println(fmt.Sprintf("Run %s. ",optList[opt]))
		fmt.Println("")
		endPoint, err := lib.GetOpt("Endpoint")
		if err != nil {
			lib.Exiter(err)
		}

		portStr, err := lib.GetOpt("Port")
		if err != nil {
			lib.Exiter(err)
		}

		port, _ := strconv.Atoi(*portStr)

		dbUsers, err := lib.GetOpt("User")
		if err != nil {
			lib.Exiter(err)
		}

		fmt.Printf("Password : ")
		vPass, err := terminal.ReadPassword(0)
		if err != nil {
			lib.Exiter(err)
		}
		var dbPass string 
		dbPass = string(vPass)
		fmt.Println("")

		db, err := lib.GetOpt("Database")
		if err != nil {
			lib.Exiter(err)
		}

		threadStr, err := lib.GetOpt("Thread Count")
		if err != nil {
			lib.Exiter(err)
		}
		thread, _ := strconv.Atoi(*threadStr)

		if thread == 0 {
			thread = 5
		} else if thread >= 10 {
			thread = 10
		}

		warmCountStr, err := lib.GetOpt("Warming Target Max Row Count")
		if err != nil {
			lib.Exiter(err)
		}

		warmCount, _ := strconv.Atoi(*warmCountStr)

		dbObj, err := lib.GetDBObject(*endPoint,port,*dbUsers,dbPass,"information_schema")
		if err != nil {
			lib.Exiter(err)
		}
		defer dbObj.Close()

		// DB Object setting
		dbObj.SetMaxIdleConns(5)
		dbObj.SetMaxOpenConns(20)
		dbObj.SetConnMaxLifetime(time.Hour)

		target, err := lib.WarmingTarget(dbObj,*db,warmCount)
		if err != nil {
			lib.Exiter(err)
		}

		var warms = make(chan string)

		// Table Warming
		for ids:=0;ids < thread;ids++ {
			go lib.Warming(dbObj,warms)
		}

		// Table In
		for _, v := range target {
			warms <- v
		}
	}
}


func ConfLoader(p string) (lib.Config, error) {
	// p : FilePath
	var c lib.Config
	yamlFile, _ := ioutil.ReadFile(p)
	err := yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}