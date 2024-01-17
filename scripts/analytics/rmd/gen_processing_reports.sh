#!/bin/bash
# Name:
#    gen_processing_reports.sh -  script for generating the f1 reports
#
# Synopsis: 
#    gen_processing_report.sh <path-to-knit> <path-to-rmd> <path-to-sqlite3-db> <output-dir>
#
# Description: 
#    Produces f1 reports in the HTML format.
#

# check the number of command line arguments
if [ "$#" -ne 4 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
knitr=$1
db=$2
rmd=$3
outputdir=$4

$knitr -d $db -f html -o f1.html -O $outputdir $rmd
