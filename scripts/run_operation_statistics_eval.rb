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

# The start of the range of blocks to be evaluated.
StartBlock = 5000000 

# The end of the range of blocks to be evaluated.
EndBlock = 50000000

# The step between block numbers at which the evaluation should be conducted
BlockStep = 1000000

# The number of transactions to be processed per evaluation point.
NumTransactionsPerEval = 100000

# The directories containing input data for Aida.
DataDir = "/var/data/aida"
SubstateDir = "#{DataDir}/substate.50M"
UpdateDir = "#{DataDir}/updateset"
DeletedAccountDir = "#{DataDir}/deleted_accounts"

# Optional extra flags to be passed to Aida.
ExtraFlags = ""

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
def runAida (startBlock) 

    puts "Running evaluation at block #{startBlock} .."
    cmd = "./build/aida-runvm --substatedir #{SubstateDir} --updatedir #{UpdateDir} --deleted-account-dir #{DeletedAccountDir} --db-impl memory --vm-impl lfvm --skip-priming --max-transactions #{NumTransactionsPerEval} --profile #{ExtraFlags} #{startBlock} 100000000"

    puts "Running #{cmd}\n"
    
    start = Time.now    
    out = ""
    Open3.popen2e(cmd) {|stdin, stdout_and_stderr, wait_thr|
    	stdout_and_stderr.each {|line|
    	        rt = (Time.now - start).to_i
    	        rt_str = "%2d:%02d:%02d" % [rt/3600,(rt%3600)/60,rt%60]
    		puts "#{DateTime.now.strftime("%Y-%m-%d %H:%M:%S.%L")} | #{rt_str} | #{startBlock} | #{line}"
                $stdout.flush
    		out.concat(line)
    	}
    }

    res = []
    out.scan(/(.*), (\d*), (.*), (.*), (.*), (.*)/) { |op,n,mean,std,min,max| res.append([op,n,mean,std,min,max]) }
    return res
end

$res = ["block, operation, n, mean, std, min, max"]
def addResult (start, stats)
    stats.each{|op, n, mean, std, min, max| $res.append("#{start}, #{op}, #{n}, #{mean}, #{std}, #{min}, #{max}") }
    $res.each{ |l| puts "#{l}\n" }
end

start = StartBlock
while start <= EndBlock
    stats = runAida(start)
    addResult(start, stats)
    start += BlockStep
end
