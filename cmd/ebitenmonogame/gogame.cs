using Microsoft.Xna.Framework;
using Microsoft.Xna.Framework.Graphics;
using Microsoft.Xna.Framework.Input;
using System;
using System.IO;

using {{.Namespace}}.AutoGen;

namespace {{.Namespace}}
{
    // Operation must sync with driver.Operation.
    enum Operation
    {
        Zero,
        One,
        SrcAlpha,
        DstAlpha,
        OneMinusSrcAlpha,
        OneMinusDstAlpha,
    }

    public class GoGame : Game
    {
        private GraphicsDeviceManager graphics;
        private IInvokable onUpdate;
        private IInvokable onDraw;
        private VertexBuffer vertexBuffer;
        private IndexBuffer indexBuffer;
        private Effect effect;

        public GoGame(IInvokable onUpdate, IInvokable onDraw)
        {
            this.onUpdate = onUpdate;
            this.onDraw = onDraw;
            this.graphics = new GraphicsDeviceManager(this);
            this.graphics.PreferredBackBufferWidth = 640;
            this.graphics.PreferredBackBufferHeight = 480;

            this.Content.RootDirectory = "Content";
            this.IsMouseVisible = true;
        }

        protected override void LoadContent()
        {
            VertexElement[] elements = new VertexElement[]
            {
                new VertexElement(sizeof(float)*0, VertexElementFormat.Vector2, VertexElementUsage.Position, 0),
                new VertexElement(sizeof(float)*2, VertexElementFormat.Vector2, VertexElementUsage.TextureCoordinate, 0),
                new VertexElement(sizeof(float)*4, VertexElementFormat.Vector4, VertexElementUsage.TextureCoordinate, 1),
                new VertexElement(sizeof(float)*8, VertexElementFormat.Vector4, VertexElementUsage.Color, 0),
            };
            this.vertexBuffer = new DynamicVertexBuffer(
                this.GraphicsDevice, new VertexDeclaration(elements), 65536, BufferUsage.None);
            this.GraphicsDevice.SetVertexBuffer(this.vertexBuffer);

            this.indexBuffer = new DynamicIndexBuffer(
                this.GraphicsDevice, IndexElementSize.SixteenBits, 65536, BufferUsage.None);
            this.GraphicsDevice.Indices = this.indexBuffer;
            
            // TODO: Add more shaders for e.g., linear filter.
            this.effect = Content.Load<Effect>("Shader");

            this.GraphicsDevice.RasterizerState = new RasterizerState()
            {
                CullMode = CullMode.None,
            };

            base.LoadContent();
        }

        internal void SetDestination(RenderTarget2D renderTarget2D, int viewportWidth, int viewportHeight)
        {
            this.GraphicsDevice.SetRenderTarget(renderTarget2D);
            this.GraphicsDevice.Viewport = new Viewport(0, 0, viewportWidth, viewportHeight);
            this.effect.Parameters["ViewportSize"].SetValue(new Vector2(viewportWidth, viewportHeight));
        }

        internal void SetSource(Texture2D texture2D)
        {
            this.effect.Parameters["Texture"].SetValue(texture2D);
        }

        internal void SetVertices(byte[] vertices, byte[] indices)
        {
            this.vertexBuffer.SetData(vertices, 0, vertices.Length);
            this.indexBuffer.SetData(indices, 0, indices.Length);
        }

        protected override void Update(GameTime gameTime)
        {
            this.onUpdate.Invoke(null);
            base.Update(gameTime);
        }

        protected override void Draw(GameTime gameTime)
        {
            this.onDraw.Invoke(null);
            base.Draw(gameTime);
        }

        internal void DrawTriangles(int indexLen, int indexOffset, Blend blendSrc, Blend blendDst)
        {
            this.GraphicsDevice.BlendState = new BlendState()
            {
                AlphaSourceBlend = blendSrc,
                ColorSourceBlend = blendSrc,
                AlphaDestinationBlend = blendDst,
                ColorDestinationBlend = blendDst,
            };
            foreach (EffectPass pass in this.effect.CurrentTechnique.Passes)
            {
                pass.Apply();
                this.GraphicsDevice.DrawIndexedPrimitives(
                    PrimitiveType.TriangleList, 0, indexOffset, indexLen / 3);
            }
        }
    }

    // This methods are called from Go world. They must not have overloads.
    class GameGoBinding
    {
        private static Blend ToBlend(Operation operation)
        {
            switch (operation)
            {
                case Operation.Zero:
                    return Blend.Zero;
                case Operation.One:
                    return Blend.One;
                case Operation.SrcAlpha:
                    return Blend.SourceAlpha;
                case Operation.DstAlpha:
                    return Blend.DestinationAlpha;
                case Operation.OneMinusSrcAlpha:
                    return Blend.InverseSourceAlpha;
                case Operation.OneMinusDstAlpha:
                    return Blend.InverseDestinationAlpha;
            }
            throw new ArgumentOutOfRangeException("operation", operation, "");
        }

        private GoGame game;

        private GameGoBinding(IInvokable onUpdate, IInvokable onDraw)
        {
            this.game = new GoGame(onUpdate, onDraw);
        }

        private void Run()
        {
            try
            {
                this.game.Run();
            }
            finally
            {
                this.game.Dispose();
            }
        }

        private RenderTarget2D NewRenderTarget2D(double width, double height)
        {
            return new RenderTarget2D(this.game.GraphicsDevice, (int)width, (int)height);
        }

        private void Dispose(Texture2D texture2D)
        {
            texture2D.Dispose();
        }

        private void ReplacePixels(RenderTarget2D renderTarget2D, byte[] pixels, double x, double y, double width, double height)
        {
            var rect = new Rectangle((int)x, (int)y, (int)width, (int)height);
            renderTarget2D.SetData(0, rect, pixels, 0, pixels.Length);
        }

        private byte[] Pixels(Texture2D texture2D, double width, double height)
        {
            Rectangle rect = new Rectangle(0, 0, (int)width, (int)height);
            byte[] data = new byte[4*(int)width*(int)height];
            texture2D.GetData(0, rect, data, 0, data.Length);
            return data;
        }

        private void SetDestination(RenderTarget2D renderTarget2D, double viewportWidth, double viewportHeight)
        {
            this.game.SetDestination(renderTarget2D, (int)viewportWidth, (int)viewportHeight);
        }

        private void SetSource(Texture2D texture2D)
        {
            this.game.SetSource(texture2D);
        }

        private void SetVertices(byte[] vertices, byte[] indices)
        {
            this.game.SetVertices(vertices, indices);
        }

        private void Draw(double indexLen, double indexOffset, double operationSrc, double operationDst)
        {
            this.game.DrawTriangles((int)indexLen, (int)indexOffset, ToBlend((Operation)operationSrc), ToBlend((Operation)operationDst));
        }

        private bool IsKeyPressed(string driverKey)
        {
            Keys key;
            switch (driverKey)
            {
                case "KeyDown":
                    key = Keys.Down;
                    break;
                case "KeyLeft":
                    key = Keys.Left;
                    break;
                case "KeySpace":
                    key = Keys.Space;
                    break;
                case "KeyRight":
                    key = Keys.Right;
                    break;
                case "KeyUp":
                    key = Keys.Up;
                    break;
                default:
                    return false;
            }
            return Keyboard.GetState().IsKeyDown(key);
        }
    }
}
