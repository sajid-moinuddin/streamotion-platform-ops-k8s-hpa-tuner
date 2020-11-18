pipeline {
    agent {
        label "streamotion-docker-in-docker"
    }

    environment {
        ORG = 'fsa-streamotion'
        APP_NAME = 'streamotion-platform-ops-k8s-hpa-tuner'
    }

    stages {
        stage('Unit-Test') {
            steps {
                container('generic') {
                    sh "make unit-tests"
                }
            }
        }
        stage('Integration-Test') {
            when {
                branch 'PR-*'
            }
            environment {
                /*in the dind container, jenkins-k8s plugin mounts the default /var/run/docker.sock to the k8s node host docker daemon port, lets not mess with that
                * instead the DOCKER im gonna run in this POD will use /var/run/dind.sock to publish docker daemon api*/
                DOCKER_HOST = "unix:///var/run/dind.sock"
                KUBECONFIG = "$HOME/.kube/config"

                PREVIEW_VERSION = "0.0.0-SNAPSHOT-$PREVIEW_NAMESPACE-$BUILD_NUMBER"
            }
            steps {
                container('dind') {
                    sh "env"
                    sh "whoami"
                    sh "echo $HOME"
                    retry(3) { //flacky docker pulls
                        sh 'kill -SIGTERM "$(pgrep dockerd)" || echo "NO dockerd found"'
                        sh "sleep 5"
                        sh "/usr/bin/dockerd -H unix:///var/run/dind.sock &"
                        sh 'sleep 15' //wait for docker to be ready
                        sh "docker ps"
                        sh 'rm -rf $HOME/.kube/config | echo "No previous Kubeconfig found"'


                        sh "kubectl get po -A"

                        sh 'make kind-delete | echo "No Clusters found"'
                        sh "sleep 10"
                        sh "make kind-test-setup"
                    }
                    sh "sleep 10"
                    sh "kubectl get po -A"
                    sh "kind get clusters"
                    sh 'make kind-tests | sleep 6000'

                }
            }
            post {

                failure {
                    //kill the docker engine
                    sh "echo FAILED!!! Pls see POD logs"
                    sh "kubectl get po -n phpload"
                    sh "kubectl get hpa -o yaml -n phpload"
                    sh "kubectl get hpatuner -o yaml -n phpload"

                    sh "sleep 6000"
//                    sh 'kill -SIGTERM "$(pgrep dockerd)" || echo "dockerd not running"'
                }


            }

        }

        stage('PREVIEW DOCKER BUILD') {
            when {
                branch 'PR-*'
            }
            environment {
                PREVIEW_NAMESPACE = "preview"
                PREVIEW_VERSION = "0.0.0-SNAPSHOT-$PREVIEW_NAMESPACE-$BUILD_NUMBER"
                HELM_RELEASE = "$APP_NAME-$BRANCH_NAME".toLowerCase()
            }
            steps {
                container('generic') {
                    sh "env"
                    sh "whoami"
                    sh "echo $HOME"
                    sh "echo **************** PREVIEW_VERSION: $PREVIEW_VERSION , PREVIEW_NAMESPACE: $PREVIEW_NAMESPACE, HELM_RELEASE: $HELM_RELEASE"
                    sh "echo $PREVIEW_VERSION > PREVIEW_VERSION"
                    sh "export VERSION=$PREVIEW_VERSION && skaffold build -f skaffold.yaml"
                }
            }
        }

        stage('Test the helm charts') {
            when {
                branch 'PR-*'

            }

            environment {
                PREVIEW_NAMESPACE = "preview"
                DOCKER_HOST = "unix:///var/run/dind.sock"
                KUBECONFIG = "$HOME/.kube/config"
                PREVIEW_VERSION = "0.0.0-SNAPSHOT-$PREVIEW_NAMESPACE-$BUILD_NUMBER"
            }

            steps {
                container('dind') {
                    sh "kubectl get po -A"
                    sh "kind get clusters"
                    dir('charts/preview') {
                        sh "make preview"
                        sh "kubectl create namespace preview"
                        sh "jx ns preview"
                        sh "helm template --namespace preview --name hpa-tuner . | kubectl apply -f -"
                        retry(5) {
                            sh "sleep 10"

                            sh "kubectl get po -n preview"
                            sh "kubectl get deploy -n preview"

                            sh "kubectl get deploy hpa-tuner-preview  --output=jsonpath={.status.readyReplicas} | grep -q '1'"
                        }
                    }
                    echo "...... cleanup resources from integration tests ......"
                    sh "kubectl delete -f config/samples/hpa-php-load.yaml"
                    sh "kubectl delete -f config/samples/webapp_v1_hpatuner.yaml"

                    sh "sleep 15"

                    sh "kubectl apply -f config/samples/hpa-php-load.yaml"
                    sh "kubectl apply -f config/samples/webapp_v1_hpatuner.yaml"
                    sh "kubectl get hpa -n phpload "

                    retry(5) {
                        sh "sleep 15"
                        echo """
                                THE TEST: the hpa was created with spec.minReplicas = 1, if its turned to 10, that means hpa-tuner controller was
                                successfully deployed and changed it to 10
                             """
                        sh "kubectl describe hpa -n phpload"
                        sh "kubectl describe hpatuner -n phpload"

                        sh "kubectl get hpa -n phpload php-apache --output=jsonpath={.spec.minReplicas}"
                        sh "kubectl get hpa -n phpload php-apache --output=jsonpath={.spec.minReplicas} | grep 10"
                    }
                }
            }

            post {
                failure {
                    //kill the docker engine
                    sh "FAILED!!! Pls see POD logs"
                    sh "sleep 600"
//                    sh 'kill -SIGTERM "$(pgrep dockerd)" || echo "dockerd not running"'
                }
            }

        }

        stage('Push To ECR') {
            when {
                branch 'master'
            }
            steps {
                container('generic') {

                    // ensure we're not on a detached head
                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"

                    sh "echo \$(jx-release-version) > VERSION"
                    sh "jx step tag --version \$(cat VERSION)"
                    sh "skaffold version"
                    sh "export VERSION=`cat VERSION` && skaffold build -f skaffold.yaml"
                    sh "export VERSION=latest && skaffold build -f skaffold.yaml"
                    script {
                        def buildVersion = readFile "${env.WORKSPACE}/VERSION"
                        currentBuild.description = "${DOCKER_REGISTRY}/streamotion-platform-ops-k8s-hpa-tuner:$buildVersion"
                        currentBuild.displayName = "$buildVersion"
                    }
                }
            }
        }

        stage('release') {
            when {
                branch 'master'
            }
            steps {
//        container('generic') {
                container('generic') {
                    sh "mv charts/helm-release  charts/$APP_NAME"
                    dir("charts/$APP_NAME") {
                        sh "jx step changelog --generate-yaml=false --version v\$(cat ../../VERSION)"

                        sh "make release"

                        sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=hpatuner-staging --version \$(cat ../../VERSION)"
                    }
                }
            }
        }
    }
    post {
        always {
            cleanWs()
        }
    }
}

def String getStackTrace(Throwable aThrowable) {
    ByteArrayOutputStream baos = new ByteArrayOutputStream();
    PrintStream ps = new PrintStream(baos, true);
    aThrowable.printStackTrace(ps);
    return baos.toString();
}