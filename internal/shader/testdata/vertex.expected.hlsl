cbuffer Uniforms : register(b0) {
	float2 U0 : packoffset(c0);
}

Varyings VSMain(float2 A0 : POSITION, float2 A1 : TEXCOORD, float4 A2 : COLOR) {
	Varyings varyings;
	float4x4 l0 = 0.0;
	varyings.Position = 0.0;
	varyings.M0 = 0.0;
	varyings.M1 = 0.0;
	l0 = float4x4((2.0) / ((U0).x), 0.0, 0.0, 0.0, 0.0, (2.0) / ((U0).y), 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -1.0, -1.0, 0.0, 1.0);
	varyings.Position = mul(float4(A0, 0.0, 1.0), l0);
	varyings.M0 = A1;
	varyings.M1 = A2;
	return varyings;
}
