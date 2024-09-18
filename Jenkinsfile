pipeline {
    agent { label 'quick' }

    options {
        timestamps ()
        timeout(time: 1, unit: 'HOURS')
        disableConcurrentBuilds(abortPrevious: true)
    }

    environment {
        GOROOT = '/usr/lib/go-1.21/'
        STORAGE = '--db-impl carmen --db-variant go-file --carmen-schema 3'
        PRIME = '--update-buffer-size 4000'
        VM = '--vm-impl lfvm'
        AIDADB = '--aida-db=/mnt/aida-db-central/aida-db'
        TMPDB = '--db-tmp=/mnt/tmp-disk'
        DBSRC = '/mnt/tmp-disk/state_db_carmen_go-file_${TOBLOCK}'
        TRACEDIR = 'tracefiles'
        FROMBLOCK = 'opera'
        TOBLOCK = '4600000'
    }

    stages {
        stage('Validate commit') {
            steps {
                script {
                    def CHANGE_REPO = sh (script: "basename -s .git `git config --get remote.origin.url`", returnStdout: true).trim()
                    build job: '/Utils/Validate-Git-Commit', parameters: [
                        string(name: 'Repo', value: "${CHANGE_REPO}"),
                        string(name: 'Branch', value: "${env.CHANGE_BRANCH}"),
                        string(name: 'Commit', value: "${GIT_COMMIT}")
                    ]
                }
            }
        }

        stage('Run tests') {
            stages {
                stage('Check formatting') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh '''diff=`find . \\( -path ./carmen -o -path ./tosca \\) -prune -o -name '*.go' -exec gofmt -s -l {} \\;`
                                  echo $diff
                                  test -z $diff
                               '''
                        }
                    }
                }

                stage('Build') {
                    steps {
                        script {
                            currentBuild.description = "Building on ${env.NODE_NAME}"
                        }
                        sh "git submodule update --init --recursive"
                        sh "make all"
                    }
                }

                stage('Run unit tests') {
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

                stage('aida-vm-sdb s5-archive+validate-state-hash') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} --db-impl carmen --validate-state-hash --archive --archive-variant s5 --carmen-schema 5 --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb validate-tx') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} --db-impl carmen --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb archive-inquirer') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} --archive --archive-query-rate 5000 --db-impl carmen --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb keep-db') {
                    steps {
                        sh "rm -rf ${DBSRC}"
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} --keep-db --archive --archive-variant ldb --db-impl carmen --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb db-src') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} --db-src ${DBSRC} ${AIDADB} --validate-tx --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure 4600001 4610000"
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-adb validate-tx') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-adb ${AIDADB} --db-src ${DBSRC} --cpu-profile cpu-profile.dat --validate-tx ${FROMBLOCK} ${TOBLOCK}"
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
        }
    }
}
