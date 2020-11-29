package main

import (
	"github.com/vivekthangathurai/demo-terraform-provider/kubetest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"	
)


func main() {
	
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubetest.Provider})
}


  



  
  