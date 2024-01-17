#!/bin/bash
# Name:
#    gen_processing_reports.sh -  script for generating the f1 reports
#
# Synopsis: 
#    gen_processing_report.sh <path-to-rmd> <path-to-sqlite3-db> <output-dir>
#
# Description: 
#    Produces f1 reports in the HTML format.
#

# check the number of command line arguments
if [ "$#" -ne 3 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
db=$1
rmd=$2
outputdir=$3

log "render f1 report ..."
./knit.R -d $db -f html -o f1.html -O $outputdir $rmd
