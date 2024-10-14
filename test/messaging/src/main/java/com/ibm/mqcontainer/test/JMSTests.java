/*
© Copyright IBM Corporation 2018, 2021

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

import static org.junit.jupiter.api.Assertions.*;

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

import com.ibm.mq.MQException;
import com.ibm.mq.constants.MQConstants;
import com.ibm.mq.jms.MQConnectionFactory;
import com.ibm.mq.jms.MQQueue;
import com.ibm.msg.client.wmq.WMQConstants;
import com.ibm.msg.client.jms.DetailedJMSSecurityRuntimeException;

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
        TrustManagerFactory tmf=TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
        tmf.init(ts);
        SSLContext ctx = SSLContext.getInstance("TLSv1.2");
        ctx.init(null, tmf.getTrustManagers(), null);
        return ctx.getSocketFactory();
    }

    static MQConnectionFactory createMQConnectionFactory(String channel, String addr) throws JMSException, IOException, GeneralSecurityException {
        MQConnectionFactory factory = new MQConnectionFactory();
        factory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
        factory.setChannel(channel);
        factory.setConnectionNameList(String.format("%s(1414)", addr));
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
                factory.setSSLCipherSuite("*TLS12ORHIGHER");
            } else {
                 System.setProperty("com.ibm.mq.cfg.useIBMCipherMappings", "false");
                 factory.setSSLCipherSuite("*TLS12ORHIGHER");
            }
        }
        return factory;
    }

    /**
     * Create a JMSContext with the supplied user and password.
     */
    static JMSContext create(String channel, String addr, String user, String password) throws JMSException, IOException, GeneralSecurityException {
        LOGGER.info(String.format("Connecting to %s/TCP/%s(1414) as %s", channel, addr, user));
        MQConnectionFactory factory = createMQConnectionFactory(channel, addr);
        // If a password is set, make sure it gets sent to the queue manager for authentication
        if (password != null) {
            factory.setBooleanProperty(WMQConstants.USER_AUTHENTICATION_MQCSP, true);
        }
        LOGGER.info(String.format("CSP authentication: %s", factory.getBooleanProperty(WMQConstants.USER_AUTHENTICATION_MQCSP)));
        return factory.createContext(user, password);
    }

    /**
     * Create a JMSContext with the default user identity (from the OS)
     */
    static JMSContext create(String channel, String addr) throws JMSException, IOException, GeneralSecurityException {
        LOGGER.info(String.format("Connecting to %s/TCP/%s(1414) as OS user '%s'", channel, addr, System.getProperty("user.name")));
        MQConnectionFactory factory = createMQConnectionFactory(channel, addr);
        LOGGER.info(String.format("CSP authentication: %s", factory.getBooleanProperty(WMQConstants.USER_AUTHENTICATION_MQCSP)));
        return factory.createContext();
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

    @Test
    void putGetTest(TestInfo t) throws Exception {
        context = create(CHANNEL, ADDR, USER, PASSWORD);
        Queue queue = new MQQueue("DEV.QUEUE.1");
        context.createProducer().send(queue, t.getDisplayName());
        Message m = context.createConsumer(queue).receive();
        assertNotNull(m.getBody(String.class));
    }

    @Test
    void defaultIdentityTest(TestInfo t) throws Exception {
        LOGGER.info(String.format("Password='%s'", PASSWORD));
        try {
            // Don't pass a user/password, which should cause the default identity to be used
            context = create(CHANNEL, ADDR);
        } catch (DetailedJMSSecurityRuntimeException ex) {
            Throwable cause = ex.getCause();
            assertNotNull(cause);
            assertTrue(cause instanceof MQException);
            assertEquals(MQConstants.MQRC_NOT_AUTHORIZED, ((MQException)cause).getReason());
            return;
        }
        // The default developer config allows any user to appear as "app", and use a blank password.  This is done with the MCAUSER on the channel.
        // If this test is run on a queue manager without a password set, then it should be possible to connect without exception.
        // If this test is run on a queue manager with a password set, then an exception should be thrown, because this test doesn't send a password.
        if ((PASSWORD != null) && (PASSWORD != "")) {
            fail("Exception not thrown");
        }
    }

    @AfterEach
    void tearDown() {
        if (context != null) {
            context.close();
        }
    }
}