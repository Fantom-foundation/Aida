#!/usr/bin/env Rscript

#
# Render script for R Markdown files.
#
# List of flags:
#  -d (--db)         profile DB for rendering
#  -o (--output)     output file of the renderer
#  -O (--output-dir) output directory
#  -f (--format)     output format [html|pdf|html_pdf]
#  -p (--parameter)  specify the parameters for R-Markdown [p1=x1,p2=x2,...]

# The required libraries have to be installed beforehand using R console
library(rmarkdown)
library(optparse)
library(tools)

# parse command-line arguments 
cat("Parse command line arguments...\n")
option_list <- list(
    make_option(c("-d", "--db"),         default="./data.db",       help="db for rendering experiment."),
    make_option(c("-o", "--output"),     default="./report",        help="output file"),
    make_option(c("-O", "--output-dir"), default="./",              help="output Directory"),
    make_option(c("-f", "--format"),     default="html",            help="output format [html|pdf|html_pdf]"),
    make_option(c("-p", "--parameter"),  default="",                help="parameters for rmd document [p1=x1,p2=x2,...]")
)
opt <- parse_args(OptionParser(usage="%prog [options] file", option_list=option_list), positional_argument=1)

# retrieve options and argument
cat("Retrieve options and argument\n")
db <- opt$options$db
output <- opt$options$output
outputDirectory <- opt$options$`output-dir`
outputFormat <- opt$options$format
parameter <- eval(parse(text=paste("list(",opt$options$param,")")))
parameter <- c(list( db = normalizePath(db)), parameter)
rmdFile <- opt$args 

# check output format
cat("Check output format\n")
if (outputFormat == "html") {
    docFormat = "html_document"
} else if (outputFormat == "pdf") {
    docFormat = "pdf_document"
} else if (outputFormat == "html_pdf") {
    docFormat = c("html_document", "pdf_document")
} else {
    stop(sprintf("Unknown output file format"))
}

# check whether R Markdown file exists
cat("check whether R Markdown file exists\n")
if(file.access(rmdFile)==-1) {
    stop(sprintf("R-Markdown file %s cannot be found", rmdFile))
}

# check whether the profile DB file exists
cat("check whether the profile DB file exists\n")
if(file.access(db)==-1) {
    stop(sprintf("Profile DB %s cannot be found", db))
}

# check whether output directory exists
cat("check whether output directory exists\n")
if(file.access(outputDirectory)==-1) {
    stop(sprintf("Output directory %s cannot be found", outputDirectory))
}

# render the R markdown document
cat("render the R markdown document\n")
render(
    input = rmdFile, 
    output_file = output, 
    output_dir = outputDirectory,
    output_format = docFormat,
    params =  parameter
)
