# RDS-Toolkit
It is a toolkit that supports various functions required for DBA management of RDS.

## Supported
Supported features are as follows.
- Multi Create Snapshot
- Multi Upgrade
- RDS Connection Check
- Describe RDS Version 
- RDS Cache Warmings

## Configure
toolkit-setup.yml
```yml
snapshot:
  prefix : export
  suffix : 202300
  target:
    - region : ap-northeast-2
      dblist :
        - dbname : database-a2
          type : instance
upgrade:
  - engine: aurora-mysql
    region: ap-northeast-2
    from: 5.7.mysql_aurora.2.07.8
    to: 5.7.mysql_aurora.2.11.2
    dblist:
      - dbname: database-1
        type: cluster


```
### snapshot
- prefix : The prefix when creating snapshots.
- suffix : The suffix when creating a snapshot.
- target :
  - region : region in the target DB list
  - dblist : This is a list of target DBs belonging to the region.
    - dbname : DB Cluster Identifier or DB Instance Identifier
    - type : cluster or instance(enum)

### upgrade
- engine : Database Engine
  - aurora-mysql
  - mysql
  - aurora-postgresql
  - postgres
- region : Region in the target DB list
- from : Current engine version
- to : Upgrade engine version
- dblist : This is a list of target DBs belonging to the region.
    - dbname : DB Cluster Identifier or DB Instance Identifier
    - type : cluster or instance(enum)

> You can create and use only the settings required for the function.

## Arg
- conf : configure file path
    - default : `./toolkit-setup.yml`
- log : toolkit log
    - default : `./toolkit.log`

### Example
```shell
./rds-toolkit_arm64.run -conf=./toolkit-setup.yml -log=./toolkit.log
```        

## Usage
### Option
```
Options:
0) Configure Reading(test)
1) Create RDS Snapshot
2) Describe Active DB Versions
3) DB Engine Upgrade
4) DB Connect Test
5) DB Cache Warming
```
0 is an item that checks whether the current Configure option is normally loaded, so it is excluded from the description.

### Create RDS Snapshot
#### Configure
```yml
snapshot:
  prefix : export
  suffix : 202304
  target:
    - region : ap-northeast-2
      dblist :
        - dbname : database-a2
          type : instance
```
#### Resource Check
```shell
Select Option : 1
Run Create RDS Snapshot. 
Current Configure:
===========================================================================
Prefix : export / Suffix : 202304
[CHECK]database-a2(ap-northeast-2) - available
DB : database-a2(ap-northeast-2) / Snapshot : export-database-a2-202304

Continue(Y/N) : y
```

#### Create Snapshot
```shell
Processing:
===========================================================================
[REQ]Create Snapshot export-database-a2-202304 for database-a2.
[OK]Snpashots : export-database-a2-202304 Start Time : 2023-04-17 17:08:24 EndTime : 2023-04-17 17:13:24
Process Complete.
```

### Describe Active DB Versions
```shell
Select Option : 2
Run Describe Active DB Versions. 

Engine:
0) aurora-mysql
1) aurora-postgresql
2) mysql
3) postgres
Select Engine : 0

Engine Versions:
5.7.mysql_aurora.2.07.0
5.7.mysql_aurora.2.07.1
5.7.mysql_aurora.2.07.2
5.7.mysql_aurora.2.07.3
5.7.mysql_aurora.2.07.4
5.7.mysql_aurora.2.07.5
5.7.mysql_aurora.2.07.6
5.7.mysql_aurora.2.07.7
5.7.mysql_aurora.2.07.8
```

### DB Engine Upgrade
#### Configure
```yml
upgrade:
  - engine: aurora-mysql
    region: ap-northeast-2
    from: 5.7.mysql_aurora.2.07.8
    to: 5.7.mysql_aurora.2.11.2
    dblist:
      - dbname: database-1
        type: cluster
```
```shell
Select Option : 3
Run DB Engine Upgrade. 
Version Check:
===========================================================================

Engine : aurora-mysql / Version : 5.7.mysql_aurora.2.07.1 -> 5.7.mysql_aurora.2.11.2
[Check]database-1 - false
Not Matched Engine Version.

Check Upgrade Configure.
exit status 1
```
Returns an error if the from version is different from the current RDS version.
#### Upgrade
```shell
Select Option : 3
Run DB Engine Upgrade. 
Version Check:
===========================================================================

Engine : aurora-mysql / Version : 5.7.mysql_aurora.2.07.8 -> 5.7.mysql_aurora.2.11.2
[Check]database-1 - true

Continue(Y/N) : y
*Snapshots are taken first.

Processing:
===========================================================================
[REQ]Upgrade Engine From 5.7.mysql_aurora.2.07.8 to 5.7.mysql_aurora.2.11.2 for database-1
[OK]Snpashots : upgrade-database-1-20230417 Start Time : 2023-04-17 17:29:23 EndTime : 2023-04-17 17:34:24
[OK]DB : database-1 Engine : 5.7.mysql_aurora.2.11.2 Start Time : 2023-04-17 17:34:24 EndTime : 2023-04-17 18:04:25
Process Complete.
```
Snapshots are created first during upgrade, and the snapshots created during upgrade follow the rules below.
- `upgrade-{cluster/instance name}-{YYYYMMDD}`

The upgrade task checks the success of the upgrade every 10 minutes.

### DB Connect Test
```shell
Select Option : 4
Run DB Connect Test. 

Engine:
0) mysql
1) postgres
Select Engine : 0
Endpoint : test1.cluster-xxxxxxxxxxxxx.ap-northeast-2.rds.amazonaws.com
Port : 3306
User :  admin
Password : 
Database : mysql
[OK]kdbadm Connect for test1.cluster-xxxxxxxxxxxxx.ap-northeast-2.rds.amazonaws.com.
```

### DB Cache Warming
When the data in the DB buffer pool is initialized after an operation such as upgrade or maintenance, all queries are performed slowly before data is loaded into memory.

#### Select Target Table 
```sql
SELECT 
	concat(table_schema,".",table_name)
FROM information_schema.tables 
WHERE TABLE_SCHEMA = ?
	and table_rows != 0
	and table_rows <= ? // Warming Target Max Row Count
	and TABLE_TYPE = 'BASE TABLE'
order by table_rows desc
```
Added Warming Target Max Row Count item. The item targets tables that are equal to or less than the entered Row Count based on statistical data.

#### Execute
```shell
Select Option : 5
Run DB Cache Warming. 

Endpoint : test1.cluster-xxxxxxxxxxxxx.ap-northeast-2.rds.amazonaws.com
Port : 3306
User : adm
Password : 
Database : localdb
Thread Count : 5
T:localdb.t1
C:14111
D:20.480834ms
====================================
T:localdb.t2
C:643
D:41.185625ms
====================================
T:localdb.t3
C:1208
D:71.48925ms
====================================
T:localdb.t4
C:140
D:10.003084ms
====================================
```