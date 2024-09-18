pipeline {
    agent { label 'quick' }

    options {
        timestamps ()
        timeout(time: 1, unit: 'HOURS')
        disableConcurrentBuilds(abortPrevious: true)
    }

    environment {
        // Aida CLI options
        STORAGE = '--db-impl carmen --db-variant go-file --carmen-schema 5'
        ARCHIVE = '--archive --archive-variant s5'
        PRIME = '--update-buffer-size 4000'
        VM = '--vm-impl lfvm'
        AIDADB = '--aida-db /mnt/aida-db-central/aida-db'
        TMPDB = '--db-tmp /mnt/tmp-disk'
        DBSRC = '/mnt/tmp-disk/state_db_carmen_go-file_${TOBLOCK}'
        PROFILE = '--cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown'

        // Other parameters
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
                            sh "build/aida-stochastic-sdb replay ${STORAGE} ${TMPDB} --db-shadow-impl geth 50 stochastic/data/simulation_uniform.json"
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
                            sh "build/aida-sdb replay ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} ${PROFILE} --shadow-db --db-shadow-impl geth --trace-file ${TRACEDIR}/trace-0.dat ${FROMBLOCK} ${TOBLOCK}"
                            sh "build/aida-sdb replay ${VM} ${STORAGE} ${TMPDB} ${AIDADB} ${PRIME} ${PROFILE} --trace-dir ${TRACEDIR} ${FROMBLOCK} ${TOBLOCK}"
                        }
                        sh "rm -rf ${TRACEDIR}"
                    }
                }

                stage('aida-vm-sdb live mode') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} ${STORAGE} --validate-tx --validate-state-hash --cpu-profile cpu-profile.dat --memory-profile mem-profile.dat --memory-breakdown --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb archive mode') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} ${STORAGE} ${ARCHIVE} ${PROFILE} --keep-db --validate-tx --validate-state-hash --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
                        }
                        sh "rm -rf *.dat"
                    }
                }

                stage('aida-vm-sdb archive-inquirer') {
                    steps {
                        catchError(buildResult: 'FAILURE', stageResult: 'FAILURE', message: 'Test Suite had a failure') {
                            sh "build/aida-vm-sdb substate ${VM} ${AIDADB} ${PRIME} ${TMPDB} ${ARCHIVE} ${PROFILE} --archive-query-rate 5000 --validate-tx --continue-on-failure ${FROMBLOCK} ${TOBLOCK} "
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
