#!/usr/bin/env  Rscript
#./tx_steady_state.R /path/to/profile.db/directory

# load libraries
library(dplyr)
library(RSQLite)


# loadArgs reads cli arguments and check if profile.db exists
loadArgs <- function(args) {
	if (length(args) != 1) {
		stop("path to profile.db must be supplied.")
	}
	path <- args[1]
	db <- paste(path, "profile.db", sep="/")
	if (!file.exists(db)) {
		stop("database not found.")
	}
	return(list(path, db))
}

# computeQuantiles function takes dataframe and a probability function as inputs,
# and returns quantile in dataframe format
computeQuantiles <- function(x, probs, na.rm =F, names =F, type = 7, ...){
	# compute quantiles
	z <- quantile(x, probs, na.rm, names, type)
	# create a data frame
	df <- data.frame(gas_quantile = probs, gas_max = z)
	# add min value
	df$gas_min <- lag(df$gas_max, n=1)
	# add bucket id
	df$gas_classifier <- as.integer(row.names(df))-1
	# reorder columns
	df <- df[,c(4,1,3,2)]
	# replace na with 0.0
	df[is.na(df)] = 0.0
	return(df)
}

# read cli arguments
ret  <- loadArgs(commandArgs(trailingOnly = TRUE))
path   <- ret[[1]]
db <- ret[[2]]
# variables
numBuckets <- 10

print("Connect to a database...")
# connect to sqlite database
con <- dbConnect(SQLite(), db)
# create a dataframe from a table
df <- dbReadTable(con, "txProfile")

# classify tx by gas consumption
df$gas_classifier <- ntile(df$gas, numBuckets)

print("Compute steady states...")
# compute steady states for each (txtype, gas classifier) pair
steadyStateDf <- df %>% count(txType, gas_classifier)
steadyStateDf$probs <- steadyStateDf$n / sum(steadyStateDf$n)

#add quantile information
quantileDf <- computeQuantiles(df$gas, 0:numBuckets/numBuckets)
steadyStateDf <- steadyStateDf %>% inner_join(quantileDf, by="gas_classifier")
steadyStateDf <- steadyStateDf[,c(1,2,5,6,7,3,4)]
# save to a csv file
outpath <- paste(path, "tx_transitions.csv", sep="/")
print(paste("Write output to", outpath))
write.csv(steadyStateDf, outpath, row.names = FALSE)
