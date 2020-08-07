pipeline {
  agent {
      label "streamotion-maven"
  }

  environment {
    ORG = 'fsa-streamotion'
    APP_NAME = 'streamotion-platform-ops-k8s-hpa-tuner'
  }

  stages {
    stage('Push To ECR') {
      steps {
        container('maven') {

          // ensure we're not on a detached head
          sh "git config --global credential.helper store"
          sh "jx step git credentials"

          sh "echo \$(jx-release-version) > VERSION"
          sh "jx step tag --version \$(cat VERSION)"
          sh "skaffold version"
          sh "export VERSION=`cat VERSION` && skaffold build -f skaffold.yaml"
          sh "export VERSION=latest && skaffold build -f skaffold.yaml"

          sleep 1000 * 60 * 30
          script {
            def buildVersion =  readFile "${env.WORKSPACE}/VERSION"
            currentBuild.description = "${DOCKER_REGISTRY}/streamotion-platform-ops-k8s-hpa-tuner:$buildVersion"
            currentBuild.displayName = "$buildVersion"
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
