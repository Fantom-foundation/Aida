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
        DBSRC = '--db-src=/var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_${TOBLOCK}'
        TRACEDIR = 'tracefiles'
        FROMBLOCK = 'opera'
        TOBLOCK = '4600000'
    }

    parameters {
        string(defaultValue: "develop", description: 'Which branch?', name: 'BRANCH_NAME')
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
                    sh "build/aida-vm ${VM} ${AIDADB} --cpu-profile cpu-profile.dat --workers 32 --validate-tx ${FROMBLOCK} ${TOBLOCK}"
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
                    sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} --db-impl carmen --validate-state-hash --archive --archive-variant s5 --carmen-schema 5 --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-sdb archive-inquirer') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} --archive --archive-query-rate 5000 --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-sdb keep-db') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb substate ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} --keep-db --archive --archive-variant ldb --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-sdb db-src') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-sdb substate ${VM} ${DBSRC} ${AIDADB} --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure 4600001 4610000"
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-adb validate-tx') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-adb ${AIDADB} ${DBSRC} --cpu-profile cpu-profile.dat --validate-tx ${FROMBLOCK} ${TOBLOCK}"
                }
                sh "rm -rf *.dat"
            }
        }

        stage('aida-vm-adb validate-tx') {
            steps {
                catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                    sh "build/aida-vm-adb ${AIDADB} ${DBSRC} --cpu-profile cpu-profile.dat --validate-tx ${FROMBLOCK} ${TOBLOCK}"
                }
                sh "rm -rf *.dat"
            }
        }


        stage('tear-down') {
            steps {
                sh "make clean"
                sh "rm -rf *.dat ${TRACEDIR}"
                sh "rm -rf /var/opera/Aida/dbtmpjenkins/state_db_carmen_go-file_${TOBLOCK}"
            }
        }
    }

    post {
        always {
            script {
                if( params.BRANCH_NAME == 'develop' ){
                    build job: '/Notifications/slack-notification-pipeline', parameters: [
                        string(name: 'result', value: "${currentBuild.result}"),
                        string(name: 'name', value: "${currentBuild.fullDisplayName}"),
                        string(name: 'duration', value: "${currentBuild.duration}"),
                        string(name: 'url', value: "$currentBuild.absoluteUrl")
                    ]
                }
            }
        }
    }
}
