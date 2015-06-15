// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io.v.webdriver.chat;

import org.junit.Test;

import io.v.webdriver.VanadiumUITestBase;
import io.v.webdriver.commonpages.OAuthLoginPage;
import io.v.webdriver.htmlreport.HTMLReportData;
import io.v.webdriver.htmlreport.HTMLReporter;

/**
 * UI tests for Vanadium Chat Web Application.
 *
 * @author alexfandrianto@google.com
 */
public class ChatUITest extends VanadiumUITestBase {
  /**
   * System property name for the test url. This will be set from the mvn command line.
   */
  private static final String PROPERTY_TEST_URL = "testUrl";

  private static final String TEST_NAME_INIT_PROCESS = "Chat Initialization Process";

  /**
   * Tests initialization process.
   * <p>
   * The process includes signing into Chrome, installing Vanadium plugin, authenticating OAuth, and
   * visiting Chat's landing page and sending a single message.
   */
  @Test
  public void testInitProcess() throws Exception {
    HTMLReportData reportData = new HTMLReportData(TEST_NAME_INIT_PROCESS, htmlReportsDir);
    curHTMLReportData = reportData;

    super.signInAndInstallExtension(reportData);

    // Get the url for the Chat web app.
    String url = System.getProperty(PROPERTY_TEST_URL);
    System.out.printf("Url: %s\n", url);
    MainPage mainPage = new MainPage(driver, url, reportData);
    if (url.equals("https://chat.staging.v.io") || url.equals("https://chat.v.io")) {
      // These are OAuth protected pages.
      OAuthLoginPage oauthLoginPage = mainPage.goToPage();
      oauthLoginPage.login();
    } else {
      mainPage.goWithoutTakingScreenshot();
    }
    super.handleCaveatTab(reportData);
    mainPage.validatePage();

    // Write html report.
    HTMLReporter reporter = new HTMLReporter(reportData);
    reporter.generateReport();
  }
}
