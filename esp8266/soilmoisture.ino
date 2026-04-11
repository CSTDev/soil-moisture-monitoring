#include <ESP8266WiFi.h>
#include <WiFiClient.h>
#include <PubSubClient.h>
#include <time.h>
#include <TZ.h>
#include <FS.h>
#include <LittleFS.h>
#include <CertStoreBearSSL.h>

#ifndef STASSID
#define STASSID "SSID" // Set your WiFi SSID
#define STAPSK "PW" // Set your WiFi Password
#define SENSOR_NAME "seedlings"
#endif

const char* ssid = STASSID;
const char* password = STAPSK;
const char* mqtt_server = "<uniqueId>.s2.eu.hivemq.cloud";
const char* sensor_name = SENSOR_NAME;

// MQTT
// A single, global CertStore which can be used by all connections.
// Needs to stay live the entire time any of the WiFiClientBearSSLs
// are present.
BearSSL::CertStore certStore;

WiFiClientSecure espClient;
PubSubClient * client;
unsigned long lastMsg = 0;
#define MSG_BUFFER_SIZE (500)
char msg[MSG_BUFFER_SIZE];
int value = 0;
// END MQTT

const int led = 13;
// 0% Dry - 622 
// 100% Water - 215
const int airValue = 622;
const int waterValue = 215;

double readInput() {
  digitalWrite(3, HIGH);
  double total = 0.0;
  for (int thisReading = 0; thisReading < 3; thisReading++) {   //take several (3) reads
    total += analogRead(A0);                                  //add them up
  }
  digitalWrite(3, LOW);
  double average = total / 3;
  double chartValue = 100 - ((average - waterValue) / (airValue - waterValue) * 100); // Convert to value between 0 and 100
  if (chartValue <= 0) {
    return 0;
  }
  if (chartValue >= 100) {
    return 100;
  }
  return chartValue;
}

void setDateTime() {
  // You can use your own timezone, but the exact time is not used at all.
  // Only the date is needed for validating the certificates.
  configTime(TZ_Europe_Berlin, "pool.ntp.org", "time.nist.gov");

  Serial.print("Waiting for NTP time sync: ");
  time_t now = time(nullptr);
  while (now < 8 * 3600 * 2) {
    delay(100);
    Serial.print(".");
    now = time(nullptr);
  }
  Serial.println();

  struct tm timeinfo;
  gmtime_r(&now, &timeinfo);
  Serial.printf("%s %s", tzname[0], asctime(&timeinfo));
}

// MQTT
void reconnect() {
  // Loop until we’re reconnected
  while (!client->connected()) {
    Serial.print("Attempting MQTT connection…");
    String clientId = "ESP8266Client - MyClient";
    // Attempt to connect
    // Insert your password
    if (client->connect(clientId.c_str(), "<mqtt username>", "<mqtt password>")) {
      Serial.println("connected");
    } else {
      Serial.print("failed, rc = ");
      Serial.print(client->state());
      Serial.println(" try again in 5 seconds");
      // Wait 5 seconds before retrying
      delay(5000);
    }
  }
}
// END MQTT

void connectWiFi() {
  if(WiFi.status() == WL_CONNECTED){
    return;
  }

  WiFi.mode(WIFI_STA);
  WiFi.begin(ssid, password);
  Serial.println("");

  // Wait for connection
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("");
  Serial.print("Connected to ");
  Serial.println(ssid);
  Serial.print("IP address: ");
  Serial.println(WiFi.localIP());
}

void setup(void) {
  pinMode(led, OUTPUT);
  digitalWrite(led, 0);
  Serial.begin(74880);

  LittleFS.begin(); // MQTT
  Serial.println("Begin");
  connectWiFi();

  // MQTT
  setDateTime();

  int numCerts = certStore.initCertStore(LittleFS, PSTR("/certs.idx"), PSTR("/certs.ar"));
  Serial.printf("Number of CA certs read: %d\n", numCerts);
  if (numCerts == 0) {
    Serial.printf("No certs found. Did you run certs-from-mozilla.py and upload the LittleFS directory before running?\n");
    return; // Can't connect to anything w/o certs!
  }

  BearSSL::WiFiClientSecure *bear = new BearSSL::WiFiClientSecure();
  // Integrate the cert store with this connection
  bear->setCertStore(&certStore);

  client = new PubSubClient(*bear);

  client->setServer(mqtt_server, 8883);

  // END MQTT
  
}

void loop(void) {
  Serial.println("Loop");
  connectWiFi();

  // MQTT
  if (!client->connected()) {
    reconnect();
  }
  client->loop();
  
  snprintf (msg, MSG_BUFFER_SIZE, "{\"name\": \"%s\", \"value\": %s}", sensor_name, String(readInput(), 2));
  Serial.print("Publish message: ");
  Serial.println(msg);
  client->publish("soilMoisture", msg);

  // END MQTT

  // Durtion to sleep in microseconds
  int count = 0;
  while(count < 2) {
    ESP.deepSleep(3.6e9);
    count++;
  }
}