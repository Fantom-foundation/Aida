#!/usr/bin/env Rscript
# TODO: Explain usage here
# This script aims to pass a list of parameters and of flags to a markdown file to yield a report for parallel transaction execution experiment.
# List of parameters:
#  Configuration
#  Processor
#  Drive
#  RepoSha
# List of flags:
#  -v (--verbose) prints verbose messages
#  -p (--parameter) parameters for R-Markdown renderer
#  -o (--output) output file
#  -f (--format) Output format [html|pdf|html_pdf]
#  -i (--input) Profile DB for rendering
# scripts/knit.r -p 'Configuration="a", Processor="ppp", Drive="ddd", RepoSha="sha"' -f html_pdf -o myreport -i ./profile.db reports/parallel_experiment.rmd
library(rmarkdown)
library(optparse)
option_list <- list(
    make_option(c("-v", "--verbose"), action="store_true", default=FALSE, help="Print verbose messages"),
    make_option(c("-p", "--parameter"), default="", help="Parameters for R-Markdown renderer"),
    make_option(c("-o", "--output"), default="./parallel.html", help="Output file"),
    make_option(c("-d", "--outputdir"), default="./", help="Output Directory"),
    make_option(c("-f", "--format"), default="html", help="Output format [html|pdf|html_pdf]"),
    make_option(c("-i", "--input"), default="./profile.db", help="Profile DB for rendering.")
)
opt <- parse_args(OptionParser(usage="%prog [options] file", option_list=option_list), positional_argument=1)
file <- opt$args 
profileDB <- opt$options$input
param <- eval(parse(text=paste("list(",opt$options$param,")")))
output <- opt$options$output
output_dir <- opt$options$outputdir

library("tools")
# checkExt checks if file extension matches expected one
checkExt <- function(file, expExtension) {
  if (file_ext(file) == expExtension) {
     stop(sprintf("file (%s) contains (%s) extension", file, expExtension))
  }
}

# Check output format
if (opt$options$format == "html") {
    format = "html_document"
    checkExt(file, "html")
} else if (opt$options$format == "pdf") {
    format = "pdf_document"
    checkExt(file, "pdf")
} else if (opt$options$format == "html_pdf") {
    format = c("html_document", "pdf_document")
    if (file_ext(file) != "") { # should file have any suffix or no suffix at all?
       stop(sprintf("file (%s) should not contain any extension", file))
    }
} else {
   stop(sprintf("R-Markdown file ( %s) cannot be found", file))
}

# TODO: Check for existence of files / directories // where ? in parameters ?


# Check input file
if(file.access(file)== -1) {
   stop(sprintf("R-Markdown file ( %s) cannot be found", file))
}

render(
  input = file, 
  output_file = output, 
  output_dir = output_dir,
  output_format = format,
  params = c(list( "ProfileDB" = profileDB), param)
)
