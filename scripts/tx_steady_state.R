#!/usr/bin/env  Rscript
# This R script computes probabilities of transaction classifier at the steady state.
# A transaction is classified by the type of transaction and amount of gas used.
# Tx Types
#	0 : regular transafer
#	1 : contract creation
#	2 : contract call
#	3 : maintenance
#./tx_steady_state.R /path/to/profile.db/directory

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

df <- loadTxProfileFromDb(db)
# classify tx by gas consumption
df$gas_classifier <- ntile(df$gas, numBuckets)

print("Compute probabilities at the steady state...")
# compute steady states for each (txtype, gas classifier) pair
steadyStateDf <- df %>% group_by(txType, gas_classifier) %>%
	summarise(n = n(), avg_gas = mean(gas))
steadyStateDf$probs <- steadyStateDf$n / sum(steadyStateDf$n)

#add quantile information
quantileDf <- computeQuantiles(df$gas, 0:numBuckets/numBuckets)
steadyStateDf <- steadyStateDf %>% inner_join(quantileDf, by="gas_classifier")
steadyStateDf <- steadyStateDf[,c(1,2,6,4,7,8,3,5)]

# save to a csv file
writeCsv(steadyStateDf, path, "tx_steady_state.csv")
