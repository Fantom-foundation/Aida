require 'open3'
require 'date'
require 'time'
require 'fileutils'

# -------------------------------- Usage --------------------------------------
#
# This script can be used to run throughput evaluations of various archive
# implementations in Carmen. The script will run different configurations in
# sequence, collecting throughput data and summarizing it at the end using a
# CSV format.
#
# To run archive evaluations, archive data is required. This can either be
# pre-recorded or created by this script before running the evaluation. To
# configure the usage of pre-recorded archive data, see the `ArchiveDirs`
# configuration parameter below.
#
# To use the script, have Aida set up in your local repository, configure the
# parameters in the following section within this script, and run the script
# in the root directory of Aida using
#
#   /path/to/aida:> ruby ./scripts/run_archive_throughput_eval.rb &> log.txt &
#   /path/to/aida:> tail -n 100 -f log.txt
#
# This will run the script in the background of your terminale (use `jobs` to 
# list background tasks and `fg` to bring it back to the foreground), while 
# allowing you to follow the progress on the terminal. The `tail` command may
# be interrupted and re-initiated as required.
#
# The script will produce its main output on the command line, which should
# thus be forwarded to a log file as outlined above. However, additionally
# CPU profile data is collected and placed in /tmp directory. A common prefix
# for those files can be specified (see the configuration below).


# ----------------------------- Configuration ---------------------------------

# Define the list of EVMs to be used. The cross-product of each EVM listed here
# and storage solution configured below will be executed.
EVMs = [
  "geth",
  "lfvm",
#  "lfvm-si",
]

# The Carmen implementation variants to be evaluated.
CARMEN_VARIANTs = [
#  "go-memory",
  "go-file",
#  "go-ldb",
#  "cpp-memory",
  "cpp-file",
#  "cpp-ldb",
]

# The Archive variants to be evaluated.
ARCHIVE_VARIANTs = [
    "ldb",
    "sql",
]

# Configure the number of parallel processing 
NUM_WORKERs = [
    1,
    2,
    4,
    6,
    8,
]

# If set to true, the evaluation will run in a loop continiously.
ENDLESS = true

# The start of the range of blocks to be evaluated.
StartBlock = 4564026 

# The end of the range of blocks to be evaluated.
#EndBlock = 5000000
EndBlock = 4600000

# An uper bound on the time spend on evaluating a single configuration before aborting it
# and moving on to the next configuration.
MaxDuration = "72h"

# The prefix to be used for CPU profile files collected.
PROFILE_FILE_PREFIX="/tmp/aida_profile_#{DateTime.now.strftime("%Y-%m-%d_%H%M%S")}"

# The directories containing input data for Aida.
#SubstateDir = "/var/data/aida/substate.50M"
DATA_DIR = "/var/data/aida"
SubstateDir = DATA_DIR + "/substate.50M"
UpdateDir = DATA_DIR + "/updateset"
DeletedAccountDir = DATA_DIR + "/deleted_accounts"

# The directory where to put recorded archive data
ARCHIVE_DIRECTORY = "./tmp_#{DateTime.now.strftime("%Y-%m-%d_%H%M%S")}"

# The directories containing archive data. Missing archives will be recorded.
ArchiveDirs = {}

# Pre-recorded archives can be configured like this:
#ArchiveDirs[["cpp-file", "ldb"]] = "/home/herbert/coding/fantom/aida/state_db_carmen_cpp-file_5000000_ldb"

# Optional extra flags to be passed to Aida.
ExtraFlags = ""

# Enable the following to enable transaction validation.
#ExtraFlags += " --validate-tx"

# ---------------------------------- Action -----------------------------------

# Step 1 - build Aida
puts "Building ... "
build_ok = system("make")
if !build_ok then
    puts "Build failed, aborting."
    exit()
end
puts "OK"

# Step 2 - record archives using various configurations
def recordArchive (db, variant, archive) 

    # Create the archive directory if it does not exist yet.
    unless File.directory?(ARCHIVE_DIRECTORY)
        FileUtils.mkdir_p(ARCHIVE_DIRECTORY)
    end

    puts "Recording archive for #{db}/#{variant}/#{archive} ..."
    cmd = "timeout #{MaxDuration} "
    cmd += "./build/aida-runvm "
    cmd += "--substatedir #{SubstateDir} "
    cmd += "--updatedir #{UpdateDir} "
    cmd += "--deleted-account-dir #{DeletedAccountDir} "
    cmd += "--db-tmp-dir #{ARCHIVE_DIRECTORY} "
    cmd += "--db-impl #{db} "
    cmd += "--db-variant \"#{variant}\" "
    cmd += "--vm-impl lfvm "
    cmd += "--archive "
    cmd += "--archive-variant #{archive} "
    cmd += "--keep-db "
    cmd += "#{StartBlock} #{EndBlock}"

    puts "Running #{cmd}\n"
    
    start = Time.now    
    out = ""
    Open3.popen2e(cmd) {|stdin, stdout_and_stderr, wait_thr|
    	stdout_and_stderr.each {|line|
    	        rt = (Time.now - start).to_i
    	        rt_str = "%2d:%02d:%02d" % [rt/3600,(rt%3600)/60,rt%60]
    		puts "#{DateTime.now.strftime("%Y-%m-%d %H:%M:%S.%L")} | recording | #{rt_str} | #{db} | #{variant} | #{archive} | #{line}"
                $stdout.flush
    		out.concat(line)
    	}
    }

    return out.match(/StateDB directory: (.*)/).captures[0]
end

DB_IMPLs = []
CARMEN_VARIANTs.each do |variant| 
    DB_IMPLs.append(["carmen",variant])
end

DB_IMPLs.each do |impl,variant|
    ARCHIVE_VARIANTs.each do |archive|
        if ArchiveDirs[[variant,archive]] == nil then
            dir = recordArchive(impl,variant,archive)
            ArchiveDirs[[variant,archive]] = dir
        end
    end
end

# Step 3 - run the archive evaluation under various configurations
def runEval(stateDirectory, evm, db, variant, archive, workers, iteration)

    puts "Running eval with #{evm} and #{db}/#{variant} with #{workers} workers .."
    cmd = "timeout #{MaxDuration} "
    cmd += "./build/aida-runarchive "
    cmd += "--substatedir #{SubstateDir} "
    cmd += "--db-src-dir #{stateDirectory} "
    cmd += "--db-impl #{db} "
    cmd += "--db-variant \"#{variant}\" "
    cmd += "--archive-variant #{archive} "
    cmd += "--vm-impl #{evm} "
    cmd += "--cpuprofile=#{PROFILE_FILE_PREFIX}_profile_#{evm}_#{db}_#{variant}_#{archive}_#{workers}_#{StartBlock}_#{EndBlock}_#{iteration}.dat "
    cmd += "--workers #{workers} "
    cmd += "#{ExtraFlags} "
    cmd += "#{StartBlock} #{EndBlock}"

    puts "Running #{cmd}\n"
    
    start = Time.now    
    out = ""
    Open3.popen2e(cmd) {|stdin, stdout_and_stderr, wait_thr|
    	stdout_and_stderr.each {|line|
    	        rt = (Time.now - start).to_i
    	        rt_str = "%2d:%02d:%02d" % [rt/3600,(rt%3600)/60,rt%60]
    		puts "#{DateTime.now.strftime("%Y-%m-%d %H:%M:%S.%L")} | eval | #{rt_str} | #{iteration} | #{evm} | #{db} | #{variant} | #{archive} | #{workers} | #{line}"
                $stdout.flush
    		out.concat(line)
    	}
    }

    rates = []
    out.scan(/Elapsed time: (.*) s, at block .* \(~ (.*) Tx\/s\)/) { |time,tx_rate| rates.append([time,tx_rate]) }
    time, tx_rate = out.match(/Total elapsed time: (.*) s, .* \(~ (.*) Tx\/s\)/).captures
    return [time, tx_rate, rates]
end

$res = ["vm, db, db-variant, archive-variant, workers, iteration, total_time, total_tx_rate"]
def addToResult (vm, db, variant, archive, workers, iteration, time, tx_rate)
    $res.append("#{vm}, #{db}, #{variant}, #{archive}, #{workers}, #{iteration}, #{time}, #{tx_rate}")
    $res.each{ |l| puts "#{l}\n" }
end

$resRates = ["vm, db, db-variant, archive-variant, workers, iteration, time, tx_rate"]
def addRatesToResult (vm, db, variant, archive, workers, iteration, rates)
    rates.each{|time,tx_rate| $resRates.append("#{vm}, #{db}, #{variant}, #{archive}, #{workers}, #{iteration}, #{time}, #{tx_rate}") }
    $resRates.each{ |l| puts "#{l}\n" }
end

iteration = 1
while true do
    DB_IMPLs.each do |impl,variant|
        ARCHIVE_VARIANTs.each do |archive|
            EVMs.each do |evm|
                NUM_WORKERs.each do |workers|
                    dir = ArchiveDirs[[variant,archive]]
                    time, tx_rate, rates = runEval(dir, evm, impl, variant, archive, workers, iteration)
                    addToResult(evm, impl, variant, archive, workers, iteration, time, tx_rate)
                    addRatesToResult(evm, impl, variant, archive, workers, iteration, rates)
                end
            end
        end
    end
    break unless ENDLESS
    iteration += 1
end
