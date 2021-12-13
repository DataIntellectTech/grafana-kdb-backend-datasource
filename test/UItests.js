const {Builder, By, until} = require('selenium-webdriver');

const grafHost = "http://localhost:3000"
const grafUser = "admin" 
const grafPass = "nondefault"  // replace with your user/pass combo if testing on seperate system
const kdbHost = "localhost"
const kdbPort = "6767"
const kdbUser = "user"
const kdbPass = "pass"
const kdbTimeout = "5000"

async function startup() {
  return await new Builder().forBrowser('chrome').build();
}

async function Login(driver) {
  await driver.get(grafHost+'/datasources/new');
  await driver.findElement(By.name("user")).sendKeys(grafUser);
  await driver.findElement(By.name("password")).sendKeys(grafPass);
  await driver.findElement(By.className("css-y3nv3e-button")).click();
}

async function NavigateNewDatasource(driver) {
  await driver.wait(until.elementLocated(By.xpath("//a[@href ='/datasources']")))
  await driver.findElement(By.xpath("//a[@href ='/datasources']")).click();
  await driver.wait(until.elementLocated(By.xpath("//a[@href ='datasources/new']")))
  await driver.findElement(By.xpath("//a[@href ='datasources/new']")).click();
  await driver.wait(until.elementLocated(By.xpath("//div[@aria-label ='Data source plugin item kdb-backend-datasource']")))
  await driver.findElement(By.xpath("//div[@aria-label ='Data source plugin item kdb-backend-datasource']")).click()
}

async function TestNewDatasource(driver) {
  await driver.wait(until.elementLocated(By.name("HostInputField")))
  await driver.findElement(By.name("HostInputField")).sendKeys(kdbHost)
  await driver.findElement(By.name("PortInputField")).sendKeys(kdbPort)
  await driver.findElement(By.name("UsernameInputField")).sendKeys(kdbUser)
  await driver.findElement(By.name("PasswordInputField")).sendKeys(kdbPass)
  await driver.findElement(By.name("TimeoutInputField")).sendKeys(kdbTimeout)
  await driver.findElement(By.className("btn btn-primary")).click()
  await driver.wait(until.elementLocated(By.className("alert-success alert")), 6000)
  var successAlerts = await driver.findElements(By.className("alert-success alert"))
  return successAlerts.length > 0
}

async function main() {
  var driver = await startup()
  await Login(driver)
  await NavigateNewDatasource(driver)
  var testres = await TestNewDatasource(driver)
  testres ? console.log("PASSED") : console.log("FAILED")
}

main()