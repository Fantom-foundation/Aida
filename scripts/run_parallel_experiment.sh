#!/bin/bash
#
# Synopsis: 
#    run_parallel_experiment.sh <aida-db> <carmen-impl> <carmen-variant> <tosca-impl> <tmp-directory> <startBlock> <endBlock> <output-directory>

#  HardwareDescription() queries the hardware configuration of the server.
#  Requires Linux as an operating system, and hwinfo must be installed.
HardwareDescription()  {
    processor=`cat /proc/cpuinfo | grep "^model name" | head -n 1 | awk -F': ' '{print $2}'`
    memory=`free | grep "^Mem:" | awk '{printf("%dGB RAM\n", $2/1024/1024)}'` 
    disks=`hwinfo  --disk | grep Model | awk -F ': \"' '{if (NR > 1) printf(", "); printf("%s", substr($2,1,length($2)-1));}  END {printf("\n")}'`
    echo "$processor, $memory, disks: $disks"
}

# OperatingSystemDescription() queries the operating system of the server.
# Requires linux as an operating system.
OperatingSystemDescription() {
    cat /etc/*-release | awk -F'=\"' '{if ($1 == "DISTRIB_DESCRIPTION") print substr($2,1,length($2)-1)}'
}

## GitHash() queries the git hash of the Aida repository
# Requires a git command installed and the AIDA repository.
# TODO: Make Aida github directory parameterisable
GitHash() {
   git rev-parse HEAD
}

## GoVersion() queries the GO version installed on the server
# Requires a local GO installation on the system.
GoVersion() {
    go version
}

# Run full parallel experiment
echo "Running profiling from startBlock $6 up endBlock $7 ..."
./build/aida-profile parallelisation --aida-db $1 --db-impl $2 --db-variant $3 --vm-impl=$4 --db-tmp $5 $6 $7 "$8/profile.db"

# Reduce dataset in sqlite3 (R is too slow / consumes too much memory)
echo "Reducing data set..."
sqlite3 $5/profile.db << EOF
-- create temporary table groupedParallelProfile to group data for every 100,000 blocks
DROP TABLE IF EXISTS  groupedParallelProfile; 
CREATE TABLE groupedParallelProfile(block INTEGER, tBlock REAL, tCommit REAL, speedup REAL);
INSERT INTO groupedParallelProfile SELECT block/100000, tBlock, tCommit, log(speedup) FROM parallelprofile;
-- aggregate data 
DROP TABLE IF EXISTS aggregatedParallelProfile;
CREATE TABLE aggregatedParallelProfile(block INTEGER, tBlock REAL, tCommit REAL, speedup REAL);
INSERT INTO aggregatedParallelProfile SELECT min(block)*100000, avg(tBlock)/1e6, avg(tCommit)/1e6, exp(avg(speedup)) FROM groupedParallelProfile GROUP BY block;
DROP TABLE groupedParallelProfile;
EOF

# Query the configuration
echo "Query configuration..."
hw=`HardwareDescription`
os=`OperatingSystemDescription`
gh=`GitHash`
go=`GoVersion`
statedb="$2 $3"
vm="$4"

# Render R Markdown file
echo "Rendering document..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', GoInfo='$go', VM='$vm', StateDB='$statedb'" -d "$8/profile.db" -f html -o parallel_experiment.html -O $8 reports/parallel_experiment.rmd 
