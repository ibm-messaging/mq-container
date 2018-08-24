/*
© Copyright IBM Corporation 2018

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package com.ibm.mqcontainer.test;

import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.io.FileInputStream;
import java.io.IOException;
import java.net.Socket;
import java.security.GeneralSecurityException;
import java.security.KeyStore;
import java.util.logging.Logger;

import javax.jms.JMSContext;
import javax.jms.JMSException;
import javax.jms.Message;
import javax.jms.Queue;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;
import javax.net.ssl.TrustManagerFactory;

import com.ibm.mq.jms.MQConnectionFactory;
import com.ibm.mq.jms.MQQueue;
import com.ibm.msg.client.wmq.WMQConstants;

import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Disabled;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.TestInfo;

class JMSTests {
    private static final Logger LOGGER = Logger.getLogger(JMSTests.class.getName());
    protected static final String ADDR = System.getenv("MQ_PORT_1414_TCP_ADDR");
    protected static final String USER = System.getenv("MQ_USERNAME");
    protected static final String PASSWORD = System.getenv("MQ_PASSWORD");
    protected static final String CHANNEL = System.getenv("MQ_CHANNEL");
    protected static final String TRUSTSTORE = System.getenv("MQ_TLS_TRUSTSTORE");
    protected static final String PASSPHRASE = System.getenv("MQ_TLS_PASSPHRASE");
    private JMSContext context;

    static SSLSocketFactory createSSLSocketFactory() throws IOException, GeneralSecurityException {
        KeyStore ts=KeyStore.getInstance("jks");
        ts.load(new FileInputStream(TRUSTSTORE), PASSPHRASE.toCharArray());
        // KeyManagerFactory kmf=KeyManagerFactory.getInstance(KeyManagerFactory.getDefaultAlgorithm());
        TrustManagerFactory tmf=TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
        tmf.init(ts);
        // tmf.init();
        SSLContext ctx = SSLContext.getInstance("TLSv1.2");
        // Security.setProperty("crypto.policy", "unlimited");
        ctx.init(null, tmf.getTrustManagers(), null);
        return ctx.getSocketFactory();
    }

    static JMSContext create(String channel, String addr, String user, String password) throws JMSException, IOException, GeneralSecurityException {
        LOGGER.info(String.format("Connecting to %s/TCP/%s(1414) as %s", channel, addr, user));
        MQConnectionFactory factory = new MQConnectionFactory();
        factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
        factory.setChannel(channel);
        factory.setConnectionNameList(String.format("%s(1414)", addr));
        // If a password is set, make sure it gets sent to the queue manager for authentication
        if (password != null) {
            factory.setBooleanProperty(WMQConstants.USER_AUTHENTICATION_MQCSP, true);
        }
        // factory.setClientReconnectOptions(WMQConstants.WMQ_CLIENT_RECONNECT);
        if (TRUSTSTORE == null) {
            LOGGER.info("Not using TLS");
        }
        else {
            LOGGER.info(String.format("Using TLS.  Trust store=%s", TRUSTSTORE));
            SSLSocketFactory ssl = createSSLSocketFactory();
            factory.setSSLSocketFactory(ssl); 
            boolean ibmjre = System.getenv("IBMJRE").equals("true");
            if (ibmjre){
                System.setProperty("com.ibm.mq.cfg.useIBMCipherMappings", "true");
                factory.setSSLCipherSuite("SSL_RSA_WITH_AES_128_CBC_SHA256");
            } else {
                 System.setProperty("com.ibm.mq.cfg.useIBMCipherMappings", "false");
                 factory.setSSLCipherSuite("TLS_RSA_WITH_AES_128_CBC_SHA256");
            }
        }
        // Give up if unable to reconnect for 10 minutes
        // factory.setClientReconnectTimeout(600);
        // LOGGER.info(String.format("user=%s pw=%s", user, password));
        return factory.createContext(user, password);
    }

    @BeforeAll
    private static void waitForQueueManager() {
        for (int i = 0; i < 20; i++) {
            try {
                Socket s = new Socket(ADDR, 1414);
                s.close();
                return;
            } catch (IOException e) {
                try {
                    Thread.sleep(500);
                } catch (InterruptedException ex) {
                }
            }
        }
    }

    @BeforeEach
    void connect() throws Exception {
        context = create(CHANNEL, ADDR, USER, PASSWORD);
    }

    @Test
    void succeedingTest(TestInfo t) throws JMSException {
        Queue queue = new MQQueue("DEV.QUEUE.1");
        context.createProducer().send(queue, t.getDisplayName());
        Message m = context.createConsumer(queue).receive();
        assertNotNull(m.getBody(String.class));
    }

    // @Test
    // void failingTest() {
    //     fail("a failing test");
    // }

    @Test
    @Disabled("for demonstration purposes")
    void skippedTest() {
        // not executed
    }

    @AfterEach
    void tearDown() {
        if (context != null) {
            context.close();
        }
    }

    @AfterAll
    static void tearDownAll() {
    }

}