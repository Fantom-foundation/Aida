#!/usr/bin/env Rscript
#
# Render script for R Markdown files.
#
# List of flags:
#  -p (--parameter) parameters for R-Markdown renderer
#  -o (--output) output file
#  -d (--outputdir) output directory
#  -f (--format) Output format [html|pdf|html_pdf]
#  -i (--input) Profile DB for rendering
#
library(rmarkdown)
library(optparse)
library("tools")

option_list <- list(
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
} else {
   stop(sprintf("R-Markdown file ( %s) cannot be found", file))
}

# Check input file
if(file.access(file)== -1) {
   stop(sprintf("R-Markdown file ( %s) cannot be found", file))
}

# Render the R markdown
render(
  input = file, 
  output_file = output, 
  output_dir = output_dir,
  output_format = format,
  params = c(list( "ProfileDB" = profileDB), param)
)
