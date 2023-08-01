#!/bin/bash
# Name:
#    run_parallel_experiment.sh -  script for running the parallel experiment for block processing
#
# Synopsis: 
#    run_parallel_experiment.sh <aida-db> <carmen-impl> <carmen-variant> <tosca-impl> <tmp-directory> <startBlock> <endBlock> <output-directory>
#
# Description: 
#    Conducts the parallel execution experiment and produces in the output directory the dataset in the form
#    of a sqlite3 database, a report in the HTML format, and the log file of the script. 
#
#    The script requires a linux environment with installed commands hwinfo, free, git, go, sqlite3, and curl.
#    The script must be invoked in the main directory of the Aida repository.
# 

# check the number of command line arguments
if [ "$#" -ne 8 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
aidadbpath=$1
dbimpl=$2
dbvariant=$3
vmimpl=$4
tmpdir=$5
startblock=$6
endblock=$7
outputdir=$8

# logging 
log() {
    echo "$(date) $1" | tee -a "$outputdir/parallel_experiment.log"
}

#  HardwareDescription() queries the hardware configuration of the server.
HardwareDescription()  {
    processor=`cat /proc/cpuinfo | grep "^model name" | head -n 1 | awk -F': ' '{print $2}'`
    memory=`free | grep "^Mem:" | awk '{printf("%dGB RAM\n", $2/1024/1024)}'` 
    disks=`hwinfo  --disk | grep Model | awk -F ': \"' '{if (NR > 1) printf(", "); printf("%s", substr($2,1,length($2)-1));}  END {printf("\n")}'`
    echo "$processor, $memory, disks: $disks"
}

# OperatingSystemDescription() queries the operating system of the server.
OperatingSystemDescription() {
    bash -lic 'source /etc/*-release; echo $DISTRIB_DESCRIPTION'
}

## GitHash() queries the git hash of the Aida repository
GitHash() {
   git rev-parse HEAD
}

## GoVersion() queries the Go version installed on the server
GoVersion() {
    go version
}

## Machine() queries machine name and IP address of server
Machine() {
    echo "`hostname`(`curl -s api.ipify.org`)"
}

# run full parallel experiment
log "run parallel experiment in the block range from $startblock to $endblock ..."
#./build/aida-profile parallelisation --aida-db $aidadbpath --db-impl $dbimpl --db-variant $dbvariant --vm-impl=$vmimpl --db-tmp $tmpdir $startblock $endblock "$outputdir/profile.db"

# reduce dataset in sqlite3 (NB: R consumes too much memory/is too slow for the reduction)
log "reduce data set ..."
sqlite3 $outputdir/profile.db << EOF
-- create temporary table groupedParallelProfile to group data for every 100,000 blocks
DROP VIEW IF EXISTS  groupedParallelProfile; 
CREATE VIEW groupedParallelProfile(block, tBlock, tCommit, numTx, speedup) AS
SELECT block/100000, tBlock, tCommit, numTx, log(speedup) FROM parallelprofile;
-- aggregate data 
DROP TABLE IF EXISTS aggregatedParallelProfile;
CREATE TABLE aggregatedParallelProfile(block INTEGER, tBlock REAL, tCommit REAL, numTx REAL,  speedup REAL);
INSERT INTO aggregatedParallelProfile SELECT min(block)*100000, avg(tBlock)/1e6, avg(tCommit)/1e6, avg(numTx), exp(avg(speedup)) FROM groupedParallelProfile GROUP BY block;
DROP VIEW groupedParallelProfile;
EOF

# query the configuration
log "query configuration ..."
hw=`HardwareDescription`
os=`OperatingSystemDescription`
machine=`Machine`
gh=`GitHash`
go=`GoVersion`
statedb="$dbimpl($dbvariant)"

# render R Markdown file
log "render document ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$8/profile.db" -f html -o parallel_experiment.html -O $8 reports/parallel_experiment.rmd

log "finished."
