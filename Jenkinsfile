pipeline {
    agent { label 'pullrequest' }
    
    options { timestamps () }
    
    environment { 
        PATH = '/usr/local/bin:/usr/bin:/bin:/usr/local/go/bin'
        STORAGE = '--db-impl carmen --db-variant go-file --carmen-schema 3'
        PRIME = '--update-buffer-size 4000'
        VM = '--vm-impl lfvm'
        AIDADB = '--aida-db=/var/opera/Aida/mainnet-data/aida-db'
        TMPDB = '--db-tmp=/var/opera/Aida/dbtmpjenkins'
        TRACEDIR = 'tracefiles'
        FROMBLOCK = 'opera'
        TOBLOCK = '4600000'
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

        stage('aida-vm') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm ${VM} --aida-db=/var/opera/Aida/mainnet-data/aida-db --cpu-profile cpu-profile.dat --workers 32 --validate-tx ${FROMBLOCK} ${TOBLOCK}"
                }
                sh "rm -rf *.dat"
            }
        }
        
        stage('aida-fuzzing') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-stochastic-sdb replay ${STORAGE} ${TMPDB} --db-shadow-impl geth 50 data/simulation_uniform.json"
                }
            }
        }

        stage('aida-sdb record') {
            steps {
                sh "mkdir -p ${TRACEDIR}"
                sh "rm -rf ${TRACEDIR}/*"
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-sdb record --cpu-profile cpu-profile-0.dat --trace-file ${TRACEDIR}/trace-0.dat ${AIDADB} ${FROMBLOCK} ${FROMBLOCK}+1000"
                    sh "build/aida-sdb record --cpu-profile cpu-profile-1.dat --trace-file ${TRACEDIR}/trace-1.dat ${AIDADB} ${FROMBLOCK}+1001 ${FROMBLOCK}+2000"
                    sh "build/aida-sdb record --cpu-profile cpu-profile-2.dat --trace-file ${TRACEDIR}/trace-2.dat ${AIDADB} ${FROMBLOCK}+2001 ${TOBLOCK}"
                }
            }
        }

        stage('aida-sdb replay') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-sdb replay ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} --shadow-db --db-shadow-impl geth --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --trace-file ${TRACEDIR}/trace-0.dat ${FROMBLOCK} ${TOBLOCK}"
                    sh "build/aida-sdb replay ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --trace-dir ${TRACEDIR} ${FROMBLOCK} ${TOBLOCK}"
                }
                sh "rm -rf ${TRACEDIR}"
            }
        }

        stage('aida-vm-sdb validate-state-hash') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB}  --validate-state-hash --db-impl carmen --db-variant go-file  --carmen-schema 5 --archive --archive-variant s5 --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-sdb') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb substate ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} --keep-db --archive --archive-variant ldb --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf *.dat"
            }
        }


        stage('aida-vm-adb') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-adb substate ${AIDADB} --db-src /var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_4600000 --cpu-profile cpu-profile.dat --validate-tx ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf /var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_4600000"
                sh "rm -rf *.dat"
            }
        }
        stage('tear-down') {
            steps {
                sh "make clean"
                sh "rm -rf *.dat ${TRACEDIR}"
            }
        }
    }
}
