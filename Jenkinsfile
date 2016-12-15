#!/usr/bin/groovy
@Library('github.com/rawlingsj/fabric8-pipeline-library@master')
def dummy
goNode{
  dockerNode{
    def v = goRelease{
      githubOrganisation = 'fabric8io'
      dockerOrganisation = 'fabric8'
      project = 'funktion-operator'
    }
  }
}