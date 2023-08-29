#!/usr/bin/env  Rscript
#./tx_transitions.R /path/to/profile.db/directory

# load libraries
library(dplyr)
library(RSQLite)

# find absoluate path and source r_utils.R
args <- commandArgs(trailingOnly = FALSE)
file.arg.name <- "--file="
script.name <- sub(file.arg.name, "", args[grep(file.arg.name, args)])
script.dirname <- dirname(script.name)
utils <- paste(getwd(),script.dirname, "utils.R",sep="/")
source(utils)

# read cli arguments
ret  <- loadArgs(commandArgs(trailingOnly = TRUE))
path  <- ret[[1]]
db <- ret[[2]]
# variables
numBuckets <- 10

# load txProfile
df <- loadTxProfileFromDb(db)

# compute gas quantiles and save them as a CSV file
quantileDf <- computeQuantiles(df$gas, 0:numBuckets/numBuckets)
# write to a csv file
writeCsv(quantileDf, path, "gas_quantiles.csv")

# add buckets
df$gas_classifier <- ntile(df$gas, numBuckets)
# get previous data
df$prev_gas_classifier <- lag(df$gas_classifier, n=1)
df$prev_txType <- lag(df$txType, n=1)

# count transitions
countingDf <- df %>% filter(!is.na(prev_txType)) %>% count(txType, gas_classifier, prev_txType, prev_gas_classifier)
# normalize ocurrences
normalizedDf <- countingDf %>% group_by(prev_txType, prev_gas_classifier) %>% mutate(probs = n/sum(n))
normalizedDf <- normalizedDf[,c(3,4,1,2,6)]
normalizedDf <- normalizedDf[with(normalizedDf, order(prev_txType, prev_gas_classifier, txType, gas_classifier)), ]

# save to a csv file
writeCsv(normalizedDf, path, "tx_transitions.csv")
