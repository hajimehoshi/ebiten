#if OPENGL
  #define SV_POSITION POSITION
  #define VS_SHADERMODEL vs_3_0
  #define PS_SHADERMODEL ps_3_0
#else
  #define VS_SHADERMODEL vs_4_0_level_9_1
  #define PS_SHADERMODEL ps_4_0_level_9_1
#endif

Texture2D Texture;
float2 ViewportSize;

SamplerState samplerState
{
  Texture = <Texture>;
  MinFilter = Point;
  MagFilter = Point;
  MipFilter = Point;
  AddressU = Clamp;
  AddressV = Clamp;
};

struct VertexShaderInput
{
  float2 Vertex : POSITION0;
  float2 TexCoord : TEXCOORD0;
  float4 TexRegion : TEXCOORD1;
  float4 Color : COLOR0;
};

struct VertexShaderOutput
{
  float4 Position : SV_POSITION;
  float2 TexCoord : TEXCOORD0;
  float4 Color : COLOR0;
};

VertexShaderOutput MainVS(in VertexShaderInput input)
{
  VertexShaderOutput output = (VertexShaderOutput)0;

  float4x4 projectionMatrix = {
    2.0 / ViewportSize.x, 0, 0, -1,
    0, -2.0 / ViewportSize.y, 0, 1,
    0, 0, 1, 0,
    0, 0, 0, 1,
  };
  output.Position = mul(projectionMatrix, float4(input.Vertex.xy, 0, 1));
  output.TexCoord = input.TexCoord;
  output.Color = input.Color.rgba;

  return output;
}

float4 MainPS(VertexShaderOutput input) : COLOR
{
  float4 c = tex2D(samplerState, input.TexCoord.xy).rgba;
  //float4 c = Texture.Sample(samplerState, input.TexCoord.xy);
  return c * input.Color.rgba;
}

technique BasicColorDrawing
{
  pass P0
  {
    VertexShader = compile VS_SHADERMODEL MainVS();
    PixelShader = compile PS_SHADERMODEL MainPS();
  }
};
