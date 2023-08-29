# loadArgs reads cli arguments and check if profile.db exists.
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
# and returns quantile in dataframe format.
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

# loadDfFromDb function takes a valid db path and loads txProfile as a dataframe.
loadTxProfileFromDb <- function(path) {
	print("Connect to a database...")
	# connect to sqlite database
	con <- dbConnect(SQLite(), db)
	# create a dataframe from a table
	df <- dbReadTable(con, "txProfile")
	dbDisconnect(con)
	return (df)
}

# writeCsv writes a given dataframe to a csv file
writeCsv <- function(df, path, outname) {
	fullpath <- paste(path, outname, sep="/")
	print(paste("Write to ", fullpath))
	write.csv(df, fullpath, row.names = FALSE)
}
