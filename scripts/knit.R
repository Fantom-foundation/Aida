#!/usr/bin/env Rscript

#
# Render script for R Markdown files.
#
# List of flags:
#  -d (--profile-db) profile DB for rendering
#  -o (--output)     output file of the renderer
#  -O (--output-dir) output directory
#  -f (--format)     output format [html|pdf|html_pdf]
#  -p (--parameter)  specify the parameters for R-Markdown [p1=x1,p2=x2,...]

# required libraries
library(rmarkdown)
library(optparse)
library(tools)

# parse command-line arguments 
option_list <- list(
    make_option(c("-d", "--profile-db"), default="./profile.db",    help="db for rendering."),
    make_option(c("-o", "--output"),     default="./parallel.html", help="output file"),
    make_option(c("-O", "--output-dir"), default="./",              help="output Directory"),
    make_option(c("-f", "--format"),     default="html",            help="output format [html|pdf|html_pdf]"),
    make_option(c("-p", "--parameter"),  default="",                help="parameters for rmd document [p1=x1,p2=x2,...]")
)
opt <- parse_args(OptionParser(usage="%prog [options] file", option_list=option_list), positional_argument=1)

# retrieve options and argument
profileDB <- opt$options$`profile-db`
output <- opt$options$output
outputDirectory <- opt$options$`output-dir`
outputFormat <- opt$options$format
parameter <- eval(parse(text=paste("list(",opt$options$param,")")))
rmdFile <- opt$args 

# checkExtension checks the extension of a filename.
checkExtension <- function(file, extension) {
    if (file_ext(file) != extension) {
        if (extension == "")  {
            stop(sprintf("file %s does not have extension %s", file, expExtension))
        } else {
            stop(sprintf("file %s must not have an extension", file, expExtension))
        }
    }
}

# check output format
if (outputFormat == "html") {
    checkExtension(output, "html")
    docFormat = "html_document"
} else if (outputFormat == "pdf") {
    checkExtension(output, "pdf")
    docFormat = "pdf_document"
} else if (outputFormat == "html_pdf") {
    checkExtension(output, "")
    docFormat = c("html_document", "pdf_document")
} else {
    stop(sprintf("Unknown output file format"))
}

# check whether R Markdown file exists
if(file.access(rmdFile)==-1) {
    stop(sprintf("R-Markdown file %s cannot be found", rmdFile))
}

# check whether the profile DB file exists
if(file.access(profileDB)==-1) {
    stop(sprintf("Profile DB %s cannot be found", profileDB))
}

# check whether output directory exists
if(file.access(outputDirectory)==-1) {
    stop(sprintf("Output directory %s cannot be found", outputDirectory))
}

# render the R markdown document
render(
    input = rmdFile, 
    output_file = output, 
    output_dir = outputDirectory,
    output_format = docFormat,
    params = c(list( ProfileDB = normalizePath(profileDB)), parameter)
)
