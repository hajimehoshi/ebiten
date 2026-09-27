Varyings VSMain(float2 A0 : POSITION, float2 A1 : TEXCOORD, float4 A2 : COLOR0) {
	Varyings varyings;
	float2 l0 = 0.0;
	float2 l1 = 0.0;
	A0 = (A0) - ((float2)(1.0));
	(A1).x = 0.0;
	(A1)[1] = ((A1)[1]) + (1.0);
	l0 = A1;
	l1 = A0;
	A0 = l0;
	A1 = l1;
	varyings.Position = float4(A0, 0.0, 1.0);
	varyings.M0 = A1;
	varyings.M1 = A2;
	return varyings;
}
