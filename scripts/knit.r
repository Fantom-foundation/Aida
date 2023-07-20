#!/usr/bin/env Rscript
library(rmarkdown)
render(
  input = './reports/parallel_experiment.rmd', 
  params = list(
    ProfileDB = './profile.db'
))