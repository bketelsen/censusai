## Open Census with Azure App Insights ##

* Follow the guide [here](https://cda.ms/H8) to create an Application Insights instance
* Export your instrumentation key as an environment variable:
    * `export APPINSIGHTS_INSTRUMENTATIONKEY=xxxxxxx`
    * I use [direnv](http://direnv.net) and MAKE SURE that my `.envrc` file is in `.gitignore`
* Build and run the LocalForwarder (Dockerfile in the root of this repo) 
    * `make build-forward`
    * `make forward` 




### Shutdown / Cleanup

Be sure to delete the App Insights Resource Group you created in the Azure Portal when you are done experimenting so you won't have a lingering billable service you're not using.


### Links and Documentation

[App Insights - Go + OpenCensus](https://cda.ms/H8)

