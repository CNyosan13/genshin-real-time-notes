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
            var lblResin = new Label { Text = "Genshin Resin Threshold (0 = Max):", AutoSize = true, Top = 20, Left = 20 };
            var lblStamina = new Label { Text = "HSR Stamina Threshold (0 = Max):", AutoSize = true, Top = 70, Left = 20 };
            var lblCharge = new Label { Text = "ZZZ Charge Threshold (0 = Max):", AutoSize = true, Top = 120, Left = 20 };
            var btnSave = new Button { Text = "Save Settings", Top = 180, Left = 20, Width = 100, Height = 30 };

            numResin.Top = 20; numResin.Left = 220; numResin.Maximum = 200;
            numStamina.Top = 70; numStamina.Left = 220; numStamina.Maximum = 240;
            numCharge.Top = 120; numCharge.Left = 220; numCharge.Maximum = 240;

            btnSave.Click += BtnSave_Click;

            this.Controls.AddRange(new Control[] { lblResin, numResin, lblStamina, numStamina, lblCharge, numCharge, btnSave });
            this.Text = "HoyoLAB Notification Settings";
            this.Size = new System.Drawing.Size(400, 280);
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
                        if (root.TryGetProperty("resin_notify_threshold", out var resin)) numResin.Value = resin.GetInt32();
                        if (root.TryGetProperty("stamina_notify_threshold", out var stamina)) numStamina.Value = stamina.GetInt32();
                        if (root.TryGetProperty("charge_notify_threshold", out var charge)) numCharge.Value = charge.GetInt32();
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
                                if (element.Name == "resin_notify_threshold" || element.Name == "stamina_notify_threshold" || element.Name == "charge_notify_threshold")
                                    continue;
                                element.WriteTo(writer);
                            }
                            writer.WriteNumber("resin_notify_threshold", (int)numResin.Value);
                            writer.WriteNumber("stamina_notify_threshold", (int)numStamina.Value);
                            writer.WriteNumber("charge_notify_threshold", (int)numCharge.Value);
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
