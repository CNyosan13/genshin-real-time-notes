using System;
using System.Linq;
using System.Windows.Forms;

namespace WebViewLogin
{
    internal static class Program
    {
        /// <summary>
        ///  The main entry point for the application.
        /// </summary>
        [STAThread]
        static void Main(string[] args)
        {
            ApplicationConfiguration.Initialize();
            // Check for --settings flag in arguments
            if (args.Length > 0 && Array.Exists(args, a => a.ToLower() == "--settings"))
            {
                Application.Run(new SettingsForm());
            }
            else
            {
                Application.Run(new Form1());
            }
        }
    }
}