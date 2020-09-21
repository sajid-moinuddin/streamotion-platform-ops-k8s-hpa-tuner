pipeline {
  agent {
      label "jenkins-go"
  }

  environment {
    ORG = 'fsa-streamotion'
    APP_NAME = 'streamotion-platform-ops-k8s-hpa-tuner'
  }

  stages {
    stage('Test') {
      when {
        branch 'PR-*'
      }
      steps {
        container('go') {
          sh "make kind-test-setup"
          sh "make kind-tests"
        }
      }
    }


    stage('Push To ECR') {
      when {
        branch 'master'
      }
      steps {
        container('streamotion-go') {

          // ensure we're not on a detached head
          sh "git config --global credential.helper store"
          sh "jx step git credentials"

          sh "echo \$(jx-release-version) > VERSION"
          sh "jx step tag --version \$(cat VERSION)"
          sh "skaffold version"
          sh "export VERSION=`cat VERSION` && skaffold build -f skaffold.yaml"
          sh "export VERSION=latest && skaffold build -f skaffold.yaml"
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
