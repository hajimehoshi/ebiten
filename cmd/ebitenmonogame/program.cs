using System;
using {{.Namespace}}.AutoGen;

namespace {{.Namespace}}
{
    public static class Program
    {
        [STAThread]
        static void Main()
        {
            Go go = new Go();
            go.Run();
        }
    }
}
