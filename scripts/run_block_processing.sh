#!/bin/bash
# Name:
#    run_block_processing.sh -  script for running the parallel experiment for block processing
#
# Synopsis: 
#    run_block_processing.sh <aida-db> <carmen-impl> <carmen-variant> <tosca-impl> <tmp-directory> <startBlock> <endBlock> <output-directory>
#
# Description: 
#    Profiles block processing including the parallel experiment, produces in the output directory the dataset in the form
#    of a sqlite3 database, reports in the HTML format, and the log file of the script. 
#
#    The script requires a linux environment with installed commands hwinfo, free, git, go, sqlite3, and curl.
#    The script must be invoked in the main directory of the Aida repository.
# 

# check the number of command line arguments
if [ "$#" -ne 8 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables from command-line 
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
    echo "$(date) $1" | tee -a "$outputdir/block_processing.log"
}

# profile block processing 
log "profile block processing from $startblock to $endblock ..."
./build/aida-profile parallelisation --aida-db $aidadbpath --db-impl $dbimpl --db-variant $dbvariant --vm-impl=$vmimpl --db-tmp $tmpdir $startblock $endblock "$outputdir/profile.db"

# produce block processing reports
log "produce processing reports ..."
./script/gen_processing_reports.sh $dbimpl $dbvariant $vmimpl $outputdir

log "finished ..."
