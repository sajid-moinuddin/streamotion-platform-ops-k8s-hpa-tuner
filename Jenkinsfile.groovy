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
      when {
        branch '*/*'
      }
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
            DOCKER_HOST="unix:///var/run/dind.sock"
            KUBECONFIG="$HOME/.kube/config"
        }
      steps {
        container('dind') {
            sh "env"
            sh "whoami"
            sh "echo $HOME"
            sh 'kill -SIGTERM "$(pgrep dockerd)" || echo "NO dockerd found"'
            sh "sleep 5"
            sh "/usr/bin/dockerd -H unix:///var/run/dind.sock &"
            sh 'sleep 15' //wait for docker to be ready
            sh "docker ps"
            sh 'rm -rf $HOME/.kube/config | echo "No previous Kubeconfig found"'
            sh 'make kind-delete | echo "No Clusters found"'
            sh "sleep 10"
            sh "make kind-test-setup"
            sh "sleep 10"
            sh "kubectl get po -A"
            sh "kind get clusters"
            sh 'make kind-tests || echo tests failed!!!'

            sh 'kill -SIGTERM "$(pgrep dockerd)"'
        }
      }
      post {
        always {
            sh 'kill -SIGTERM "$(pgrep dockerd)"'
        }
      }

    }


    stage('Push To ECR') {
      when {
        branch 'master'
      }
      steps {
//        container('streamotion-go') {
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

    stage('Promote to Environments') {
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
            // promote through all 'Staging' promotion Environments
            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=platform-bedrock --version \$(cat ../../VERSION)"
            //          sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=commerce-staging --version \$(cat ../../VERSION)"
            //            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=content-staging --version \$(cat ../../VERSION)"
            //            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=streamtech-staging --version \$(cat ../../VERSION)"

            // promote through all 'Production' promotion Environments
            //          sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=commerce-production --version \$(cat ../../VERSION)"
            // sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=content-production --version \$(cat ../../VERSION)"
            // sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=streamtech-production --version \$(cat ../../VERSION)"

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
