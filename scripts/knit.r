#!/usr/bin/env Rscript
library(rmarkdown)
if (length(args)==0) {
  stop("At least one argument must be supplied (input file).n", call.=FALSE)
}
render(
  input = './reports/parallel_experiment.rmd', 
  params = list(
    ProfileDB = args[1]
))