pipeline {
    agent { label 'short' }
    
    options { timestamps () }
    
    environment { 
        PATH = '/usr/local/bin:/usr/bin:/bin:/usr/local/go/bin'
        STORAGE = '--db-impl carmen --db-variant go-file --carmen-schema 3 --db-tmp=/var/opera/Aida/dbtmpjenkins'
        PRIME = '--update-buffer-size 4096'
        VM = '--vm-impl lfvm'
        AIDADB = '--aida-db=/var/opera/Aida/mainnet-data/aida-db'
        fromBlock = 'opera'
        toBlock = '4600000'
    }

    stages {
        stage('Build') {
            steps {
                script {
                    currentBuild.description = "Building on ${env.NODE_NAME}"
                }
                sh "git submodule update --init --recursive"
                sh "make all"
            }
        }
	stage('Test') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                     sh 'go test ./...'
                }
            }
	}

        stage('aida-vm-replay') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm replay ${VM} --aida-db=/var/opera/Aida/mainnet-data/aida-db --workers 32 ${fromBlock} ${toBlock}"
                }
            }
        }
        
        stage('aida-fuzzing') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-stochastic-sdb replay ${STORAGE} --db-shadow-impl geth 50 data/simulation_uniform.json"
                }
            }
        }
        
        stage('aida-vm-sdb') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb ${VM} ${STORAGE} ${AIDADB} ${PRIME} --keep-db --archive --archive-variant ldb ${fromBlock} ${toBlock} "
                }
            }
        }
        
        stage('aida-vm-adb') {
            steps {
                sh "rm -f *.cpuprofile *.memprofile *.log"
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-adb ${AIDADB} --db-src /var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_4600000 ${fromBlock} ${toBlock} "
                }
                sh "rm -rf /var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_4600000"
            }
        }
        stage('tear-down') {
            steps {
                sh "make clean"
            }
        }
    }
}

