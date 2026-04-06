using System;
using System.IO;
using System.Text.Json;
using System.Windows.Forms;

namespace WebViewLogin
{
    public partial class SettingsForm : Form
    {
        private string configPath = "hoyo_cookie.json";
        private NumericUpDown numResin;
        private NumericUpDown numStamina;
        private NumericUpDown numCharge;
        private NumericUpDown numRefresh;

        public SettingsForm()
        {
            InitializeComponent();
            LoadSettings();
        }

        private void InitializeComponent()
        {
            this.numResin = new NumericUpDown();
            this.numStamina = new NumericUpDown();
            this.numCharge = new NumericUpDown();
            this.numRefresh = new NumericUpDown();
            var lblResin = new Label { Text = "Genshin Resin Threshold (0 = Max):", AutoSize = true, Top = 20, Left = 20 };
            var lblStamina = new Label { Text = "HSR Stamina Threshold (0 = Max):", AutoSize = true, Top = 70, Left = 20 };
            var lblCharge = new Label { Text = "ZZZ Charge Threshold (0 = Max):", AutoSize = true, Top = 120, Left = 20 };
            var lblRefresh = new Label { Text = "Refresh Interval (seconds):", AutoSize = true, Top = 170, Left = 20 };
            var btnSave = new Button { Text = "Save Settings", Top = 230, Left = 20, Width = 100, Height = 30 };

            numResin.Top = 20; numResin.Left = 220; numResin.Maximum = 200;
            numStamina.Top = 70; numStamina.Left = 220; numStamina.Maximum = 240;
            numCharge.Top = 120; numCharge.Left = 220; numCharge.Maximum = 240;
            numRefresh.Top = 170; numRefresh.Left = 220; numRefresh.Maximum = 3600; numRefresh.Minimum = 30;

            btnSave.Click += BtnSave_Click;

            this.Controls.AddRange(new Control[] { lblResin, numResin, lblStamina, numStamina, lblCharge, numCharge, lblRefresh, numRefresh, btnSave });
            this.Text = "HoyoLAB Notification Settings";
            this.Size = new System.Drawing.Size(400, 320);
            this.FormBorderStyle = FormBorderStyle.FixedDialog;
            this.StartPosition = FormStartPosition.CenterScreen;
        }

        private void LoadSettings()
        {
            try
            {
                if (File.Exists(configPath))
                {
                    string json = File.ReadAllText(configPath);
                    using (JsonDocument doc = JsonDocument.Parse(json))
                    {
                        var root = doc.RootElement;
                        // Load dynamic max values
                        if (root.TryGetProperty("max_resin", out var maxResin) && maxResin.GetInt32() > 0) numResin.Maximum = maxResin.GetInt32();
                        if (root.TryGetProperty("max_stamina", out var maxStamina) && maxStamina.GetInt32() > 0) numStamina.Maximum = maxStamina.GetInt32();
                        if (root.TryGetProperty("max_charge", out var maxCharge) && maxCharge.GetInt32() > 0) numCharge.Maximum = maxCharge.GetInt32();

                        if (root.TryGetProperty("resin_notify_threshold", out var resin)) numResin.Value = Math.Min(resin.GetInt32(), numResin.Maximum);
                        if (root.TryGetProperty("stamina_notify_threshold", out var stamina)) numStamina.Value = Math.Min(stamina.GetInt32(), numStamina.Maximum);
                        if (root.TryGetProperty("charge_notify_threshold", out var charge)) numCharge.Value = Math.Min(charge.GetInt32(), numCharge.Maximum);
                        if (root.TryGetProperty("refresh_interval", out var refresh)) numRefresh.Value = Math.Max(numRefresh.Minimum, Math.Min(3600, refresh.GetInt32()));
                    }
                }
            }
            catch (Exception ex)
            {
                MessageBox.Show("Failed to load settings: " + ex.Message);
            }
        }

        private void BtnSave_Click(object sender, EventArgs e)
        {
            try
            {
                string json = "{}";
                if (File.Exists(configPath))
                {
                    json = File.ReadAllText(configPath);
                }

                var options = new JsonWriterOptions { Indented = true };
                using (var stream = new MemoryStream())
                {
                    using (var writer = new Utf8JsonWriter(stream, options))
                    {
                        using (JsonDocument doc = JsonDocument.Parse(json))
                        {
                            writer.WriteStartObject();
                            foreach (var element in doc.RootElement.EnumerateObject())
                            {
                                if (element.Name == "resin_notify_threshold" || element.Name == "stamina_notify_threshold" || element.Name == "charge_notify_threshold" || element.Name == "refresh_interval")
                                    continue;
                                element.WriteTo(writer);
                            }
                            writer.WriteNumber("resin_notify_threshold", (int)numResin.Value);
                            writer.WriteNumber("stamina_notify_threshold", (int)numStamina.Value);
                            writer.WriteNumber("charge_notify_threshold", (int)numCharge.Value);
                            writer.WriteNumber("refresh_interval", (int)numRefresh.Value);
                            writer.WriteEndObject();
                        }
                    }
                    File.WriteAllBytes(configPath, stream.ToArray());
                }
                MessageBox.Show("Settings saved successfully!");
                this.Close();
            }
            catch (Exception ex)
            {
                MessageBox.Show("Failed to save settings: " + ex.Message);
            }
        }
    }
}
