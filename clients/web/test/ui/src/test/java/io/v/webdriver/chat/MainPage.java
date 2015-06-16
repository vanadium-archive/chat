// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io.v.webdriver.chat;

import com.google.common.base.Function;
import com.google.common.base.Predicate;

import org.junit.Assert;
import org.openqa.selenium.By;
import org.openqa.selenium.Keys;
import org.openqa.selenium.TakesScreenshot;
import org.openqa.selenium.TimeoutException;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import io.v.webdriver.Util;
import io.v.webdriver.commonpages.OAuthLoginPage;
import io.v.webdriver.commonpages.PageBase;
import io.v.webdriver.htmlreport.HTMLReportData;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * The main page of Vanadium Chat.
 *
 * @author alexfandrianto@google.com
 */
public class MainPage extends PageBase {
  /**
   * Default timeout for each Vanadium chat message.
   */
  protected static final int CHAT_TIMEOUT = 20;

  /**
   * The google username is set when this page is constructed.
   * It is used to identify the current chat user during the test.
   */
  private static final String PROPERTY_GOOGLE_BOT_USERNAME = "googleBotUsername";
  private final String username;

  /**
   * Checks whether the specified user shows up in the members section.
   * Note: Does not confirm that this particular tab has connected, so don't
   * rely on this to work if the user is already in the chatroom from some other
   * source.
   */
  private static class CheckMember implements Predicate<WebDriver> {
    private final String user;

    public CheckMember(String user) {
      this.user = user;
    }

    @Override
    public boolean apply(WebDriver driver) {
      // The members are in a div with the 'members' class.
      // The div has an unordered list of members, and the list items will eventually
      // have the user inside of it.
      System.out.println("There are " + driver.findElements(
        By.xpath("//div[@class='members']/ul/li")).size() + " users in the chatroom.");
      List<WebElement> matchingMembers = driver.findElements(
        By.xpath("//div[@class='members']/ul/li/span[text()='" + user + "']"));

      return matchingMembers.size() > 0;
    }
  }

  /**
   * Checks whether the given user sent a specific message.
   * Note: This check is dumb, so don't rely on this to be accurate when
   * checking if a message was sent twice.
   */
  private static class CheckMessage implements Predicate<WebDriver> {
    private final String user;
    private final String message;

    public CheckMessage(String user, String message) {
      this.user = user;
      this.message = message;
    }

    @Override
    public boolean apply(WebDriver driver) {
      // The messages are in a div with the 'messages' class.
      // Each message contains multiple spans. 'sender' contains the sender's
      // username, while 'text' contains the message text.
      List<WebElement> allMessages = driver.findElements(
        By.xpath("//div[@class='messages']/div"));

      System.out.println("There are " + allMessages.size() + " messages in the chatroom.");

      for (WebElement messageElem : allMessages) {
        // sender's text must match the user.
        WebElement sender = messageElem.findElement(By.xpath("span[@class='sender' and text()='" + user + "']"));

        // text's text must match the message.
        WebElement text = messageElem.findElement(By.xpath("span[@class='text' and text()='" + message + "']"));

        // We found a matching message if both sender and text are not null.
        if (sender != null && text != null) {
          return true;
        }
      }

      return false;
    }
  }

  public MainPage(WebDriver driver, String url, HTMLReportData htmlReportData) {
    super(driver, url, htmlReportData);

    username = System.getProperty(PROPERTY_GOOGLE_BOT_USERNAME);
  }

  public OAuthLoginPage goToPage() {
    super.goWithoutTakingScreenshot();
    // The first time going to the main page, it will ask for oauth login.
    return new OAuthLoginPage(driver, htmlReportData);
  }

  public void validatePage() {
    // Verify that the user shows up in the member location.
    log("Check that user is a member.");
    checkUserIsMember();

    // Verify that the user can send a chat and see it appear.
    // Note: This check assumes the chats go through the server before showing
    // up on the client side. As of writing, this is the case.
    log("Send chat message.");
    checkMessageIsDelivered();
  }

  public void checkUserIsMember() {
    CheckMember memberChecker = new CheckMember(username);
    try {
      wait.until(memberChecker);
    } catch(TimeoutException e) {
      e.printStackTrace();
      Assert.fail(e.toString());
    }
    Util.takeScreenshot((TakesScreenshot)driver, "is-member-" + username + ".png", "Is Member? " + username, htmlReportData);
  }

  // Sends a non-obtrusive message to all chatters using the chat app.
  // Note: All users of Vanadium chat are in the same chat room, so the message
  // sent should not be spammy.
  public void checkMessageIsDelivered() {
    // First, let's find the text box where we can enter our message.
    // Since its React ID could easily change (autogenerated), we use xpath to identify it.
    WebElement input = driver.findElement(By.xpath("//div[@class='compose']/form/input"));
    String message = "Vanadium Bot says Hello!";
    input.sendKeys(message);
    Util.takeScreenshot((TakesScreenshot)driver, "message-written.png", "Message Written", htmlReportData);

    // And then send the text.
    input.sendKeys(Keys.RETURN);
    Util.takeScreenshot((TakesScreenshot)driver, "message-sent.png", "Message Sent", htmlReportData);

    // Now, wait until the text shows up on screen!
    CheckMessage messageChecker = new CheckMessage(username, message);
    try {
      new WebDriverWait(driver, CHAT_TIMEOUT).until(messageChecker);
    } catch(TimeoutException e) {
      e.printStackTrace();
      Assert.fail(e.toString());
    }
    Util.takeScreenshot((TakesScreenshot)driver, "message-received.png", "Message Received", htmlReportData);
  }
}
