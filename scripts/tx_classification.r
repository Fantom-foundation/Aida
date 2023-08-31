#!/usr/bin/env  Rscript

# load libraries
library(dplyr)
library(RSQLite)

# load arguments
args <- commandArgs(trailingOnly = TRUE)
if (length(args) != 1) {
	stop("path to profile.db must be supplied.")
}
path <- args[1]
db <- paste(path, "profile.db", sep="/")
if (!file.exists(db)) {
	stop("database not found.")
}
# variables
numBuckets <- 10

# connect to sqlite database
con <- dbConnect(SQLite(), db)
# create a dataframe from a table
df <- dbReadTable(con, "txProfile")

# compute gas quantiles and save them as a CSV file
quantilesDf <- quantile(df$gas, 0:numBuckets/numBuckets)
# write to file
qpath <- paste(path, "gas_quantiles.csv", sep="/")
print(paste("Write to ", qpath))
write.csv(quantilesDf, qpath, row.names = TRUE)

# add buckets
df$gas_classifier <- ntile(df$gas, numBuckets)
# get previous data
df$prev_gas_classifier <- lag(df$gas_classifier, n=1)
df$prev_txType <- lag(df$txType, n=1)

# count transitions
countingDf <- df %>% filter(!is.na(prev_txType)) %>% count(txType, gas_classifier, prev_txType, prev_gas_classifier)
# normalize ocurrences
normalizedDf <- countingDf %>% group_by(prev_txType, prev_gas_classifier) %>% mutate(percent = n/sum(n))
normalizedDf <- normalizedDf[,c(3,4,1,2,6)]
normalizedDf <- normalizedDf[with(normalizedDf, order(prev_txType, prev_gas_classifier, txType, gas_classifier)), ]

# write to csv
txpath <- paste(path, "tx_transitions.csv", sep="/")
print(paste("Write to ", txpath))
write.csv(normalizedDf, txpath, row.names = FALSE)
