#!/bin/bash
# Name:
#    gen_processing_reports.sh -  script for generating the block processing reports
#
# Synopsis: 
#    gen_processing_report.sh <output-dir>
#
# Description: 
#    Produces block processing reports in the HTML format.
#
#    The script requires a linux environment with installed commands hwinfo, free, git, go, sqlite3, and curl.
#    The script must be invoked in the main directory of the Aida repository.
# 

# check the number of command line arguments
if [ "$#" -ne 1 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
outputdir=$1

log "render f1 report ..."
/home/rapolt/dev/aida/scripts/analytics/rmd/knit.R \
                 -d /var/opera/Aida/tmp-rapolt/register/s5-f1-test.db \
		 -f html -o test.html \
		 -O $outputdir /home/rapolt/dev/aida/scripts/analytics/rmd/f1.rmd
