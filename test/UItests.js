/* const {Builder, By, Key, until} = require('selenium-webdriver');

(async function NewDataSource() {
  let driver = await new Builder().forBrowser('chrome').build();
  try {
    await driver.get('http://localhost:3000/datasources/new');
    //var dsNames = await driver.findElements(By.className("add-data-source-item"))
    //dsNames.findElements(By.)
    await driver.findElement(By.className("css-1ihbihm-button")).click()
    await driver.findElement(By.xpath("//*[text()='Host'])"))
    //await driver.wait(until.titleIs('webdriver - Google Search'), 1000);
  } finally {
    await driver.quit();
  }
})(); */