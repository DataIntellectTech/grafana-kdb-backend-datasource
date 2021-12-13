const {Builder, By, Key, until} = require('selenium-webdriver');

async function startup() {
  return await new Builder().forBrowser('chrome').build();
}

async function Login(driver) {
  await driver.get('http://localhost:3000/datasources/new');
  await driver.findElement(By.name("user")).sendKeys("admin");
  await driver.findElement(By.name("password")).sendKeys("nondefault");
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
  await driver.findElement(By.name("HostInputField")).sendKeys("localhost")
  await driver.findElement(By.name("PortInputField")).sendKeys("6767")
  await driver.findElement(By.name("UsernameInputField")).sendKeys("user")
  await driver.findElement(By.name("PasswordInputField")).sendKeys("pass")
  await driver.findElement(By.name("TimeoutInputField")).sendKeys("5000")
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