require 'open3'
require 'date'
require 'time'

# -------------------------------- Usage --------------------------------------
#
# This script can be used to run throughput evaluations of Tosca and Carmen 
# using the Aida infrastructure. The script will run different configurations
# in sequence, collecting throughput data and summarizing it at the end using
# a CSV format.
#
# To use the script, have Aida set up in your local repository, configure the
# parameters in the following section within this script, and run the script
# in the root directory of Aida using
#
#   /path/to/aida:> ruby ./scripts/run_throughput_eval.rb &> log.txt &
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

# Define the list of modes to be tested.
MODEs = [
    "validator",
    "archive",
]

# Define the list of EVMs to be used. The cross-product of each EVM listed here
# and storage solution configured below will be executed.
EVMs = [
  "geth",
  "lfvm",
  "lfvm-si",
  "evmzero",
]

# Define the storage solutions to be evaluted.
DB_IMPLs = [
#  "memory",  # < this is the substate-only option
#  "geth",
  #  carmen is enabled if any of the variants is enabled
]

# For Carmen based options, please enable relevant variants.
CARMEN_VARIANTs = [
#  "go-memory",
  "go-file",
#  "go-ldb",
#  "cpp-memory",
  "cpp-file",
#  "cpp-ldb",
]

# The current-state DB of Carmen can implement several different schemas.
# With this option the set of schemas to be evaluated can be configured.
CARMEN_SCHEMAs = [
#    1,   # address and key indexed, no reincarnation numbers
#    2,   # address indexed, keys not, no reincarnation numbers
    3,   # address indexed, keys not, using reincarnation numbers
]

# If set to true, the evaluation will run in a loop continiously.
ENDLESS = false

# The start of the range of blocks to be evaluated.
StartBlock = 0

# The end of the range of blocks to be evaluated.
EndBlock = 70000000

# An uper bound on the time spend on evaluating a single configuration before aborting it
# and moving on to the next configuration.
MaxDuration = "72h"

# The prefix to be used for CPU profile files collected.
PROFILE_FILE_PREFIX="/tmp/aida_profile_#{DateTime.now.strftime("%Y-%m-%d_%H%M%S")}"

# The directories containing the Aida DB.
AidaDb = "/var/data/aida-db"

# Optional extra flags to be passed to Aida.
ExtraFlags = ""

# Enable the following to enable transaction validation.
#ExtraFlags += " --validate-tx"

# ---------------------------------- Action -----------------------------------

# Step 1 - build Aida
puts "Building ... "
build_ok = system("make -j aida-vm-sdb")
if !build_ok then
    puts "Build failed, aborting."
    exit()
end
puts "OK"


# Step 2 - run Aida under various configurations
def runAida (mode, evm, db, variant, schema, iteration)

    extraFlags = ExtraFlags
    if mode == "archive" then
        extraFlags += " --archive"
    end

    if !schema then
        schema = 0
    end

    puts "Running #{mode} with #{evm} and #{db}/#{variant}/s#{schema} .."
    cmd = ""
    cmd += "timeout #{MaxDuration} "
    cmd += "./build/aida-vm-sdb --aida-db #{AidaDb} "
    cmd += "--db-impl #{db} --db-variant \"#{variant}\" --carmen-schema \"#{schema}\" "
    cmd += "--vm-impl #{evm} "
    cmd += "--track-progress "
    cmd += "--cpu-profile=#{PROFILE_FILE_PREFIX}_profile_#{mode}_#{evm}_#{db}_#{variant}_#{StartBlock}_#{EndBlock}_#{iteration}.dat "
    cmd += "#{extraFlags} "
    cmd += "#{StartBlock} #{EndBlock}"

    puts "Running #{cmd}\n"
    
    start = Time.now    
    out = ""
    Open3.popen2e(cmd) {|stdin, stdout_and_stderr, wait_thr|
    	stdout_and_stderr.each {|line|
    	        rt = (Time.now - start).to_i
    	        rt_str = "%2d:%02d:%02d" % [rt/3600,(rt%3600)/60,rt%60]
    		puts "#{DateTime.now.strftime("%Y-%m-%d %H:%M:%S.%L")} | #{rt_str} | #{iteration} | #{mode} | #{evm} | #{db} | #{variant} | s#{schema} | #{line}"
                $stdout.flush
    		out.concat(line)
    	}
    }

    res = []
    pattern = /.*Track: block (\d+), memory (\d+), disk (\d+), interval_tx_rate (\d+.\d*), interval_gas_rate (\d+.\d*), overall_tx_rate (\d+.\d*), overall_gas_rate (\d+.\d*)/
    out.scan(pattern) { |block,mem_usage,disk_usage,interval_tx_rate,interval_gas_rate,overall_tx_rate,overall_gas_rate| res.append([block,mem_usage,disk_usage,interval_tx_rate,interval_gas_rate,overall_tx_rate,overall_gas_rate]) }
    return res
end

$res = ["mode, vm, db, variant, schema, iteration, interval_end, mem_usage, disk_usage, interval_tx_rate, interval_gas_rate, overall_tx_rate, overall_gas_rate"]
def addResult (mode, vm, db, variant, schema, iteration, rates)
    rates.each{|block,mem_usage,disk_usage,interval_tx_rate,interval_gas_rate,overall_tx_rate,overall_gas_rate| 
        $res.append("#{mode}, #{vm}, #{db}, #{variant}, #{schema}, #{iteration}, #{block}, #{mem_usage}, #{disk_usage}, #{interval_tx_rate}, #{interval_gas_rate}, #{overall_tx_rate}, #{overall_gas_rate}")
    }
    $res.each{ |l| puts "#{l}\n" }
end

CARMEN_VARIANTs.each do |variant|
    CARMEN_SCHEMAs.each do |schema|
        DB_IMPLs.append(["carmen",variant,schema])
    end
end

iteration = 1
while true do
    MODEs.each do |mode|
        EVMs.each do |evm|
            DB_IMPLs.each do |impl,variant,schema|
                rates = runAida(mode, evm, impl, variant, schema, iteration)
                addResult(mode, evm, impl, variant, schema, iteration, rates)
            end
        end
    end
    break unless ENDLESS
    iteration += 1
end
