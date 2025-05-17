The main example class to see is `src/main/java/GalileoApiGateway.java`.

## 3 steps to run this app

1. Install Brew:
```
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

2. Install Maven:
```
brew install maven
```

3. Run the application
```
mvn exec:java -Dexec.mainClass="GalileoApiGateway" -Dexec.args="GALILEO_API_KEY GALILEO_API_URL"
```

Example of GALILEO_API_URL: https://api.xyz.rungalileo.io

Get the GALILEO_API_KEY by 
- Going to Galileo console home
- Click your icon (on the bottom left)
- API Keys
- Create one

