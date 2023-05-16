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
#  "geth",
  "lfvm",
#  "lfvm-si",
]

# Define the storage solutions to be evaluted.
DB_IMPLs = [
  "memory",  # < this is the substate-only option
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
    1,   # address and key indexed, no reincarnation numbers
    2,   # address indexed, keys not, no reincarnation numbers
    3,   # address indexed, keys not, using reincarnation numbers
]

# If set to true, the evaluation will run in a loop continiously.
ENDLESS = true

# The start of the range of blocks to be evaluated.
StartBlock = 4564026 

# The end of the range of blocks to be evaluated.
EndBlock = 60000000

# An uper bound on the time spend on evaluating a single configuration before aborting it
# and moving on to the next configuration.
MaxDuration = "72h"

# The prefix to be used for CPU profile files collected.
PROFILE_FILE_PREFIX="/tmp/aida_profile_#{DateTime.now.strftime("%Y-%m-%d_%H%M%S")}"

# The directories containing input data for Aida.
DATA_DIR = "/var/data/aida"
SubstateDb = DATA_DIR + "/substate.50M"
UpdateDir = DATA_DIR + "/updateset"
DeletedAccountDir = DATA_DIR + "/deleted_accounts"



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
    cmd = "timeout #{MaxDuration} ./build/aida-runvm --substate-db #{SubstateDb} --updatedir #{UpdateDir} --deleted-account-dir #{DeletedAccountDir} --db-impl #{db} --db-variant \"#{variant}\" --carmen-schema \"#{schema}\" --vm-impl #{evm} --cpuprofile=#{PROFILE_FILE_PREFIX}_profile_#{mode}_#{evm}_#{db}_#{variant}_#{StartBlock}_#{EndBlock}_#{iteration}.dat #{extraFlags} #{StartBlock} #{EndBlock}"

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
    out.scan(/Reached block (.*), last interval rate ~ (.*) Tx\/s, ~ (.*) Gas\/s/) { |block,tx_rate,gas_rate| res.append([block,tx_rate,gas_rate]) }
    return res
end

$res = ["mode, vm, db, variant, schema, iteration, interval_end, tx_rate, gas_rate"]
def addResult (mode, vm, db, variant, schema, iteration, rates)
    rates.each{|block,tx_rate,gas_rate| $res.append("#{mode}, #{vm}, #{db}, #{variant}, #{schema}, #{iteration}, #{block}, #{tx_rate}, #{gas_rate}") }
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
