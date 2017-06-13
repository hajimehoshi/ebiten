// https://github.com/technosaurus/PDMP3
// License: Public Domain

#include "pdmp3.h"

#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <unistd.h>
#include <stdlib.h>

/* Types used in the frame header */
typedef enum { /* Layer number */
  mpeg1_layer_reserved = 0,
  mpeg1_layer_3        = 1,
  mpeg1_layer_2        = 2,
  mpeg1_layer_1        = 3
}
t_mpeg1_layer;
typedef enum { /* Modes */
  mpeg1_mode_stereo = 0,
  mpeg1_mode_joint_stereo,
  mpeg1_mode_dual_channel,
  mpeg1_mode_single_channel
}
t_mpeg1_mode;
typedef struct { /* MPEG1 Layer 1-3 frame header */
  unsigned id;                 /* 1 bit */
  t_mpeg1_layer layer;         /* 2 bits */
  unsigned protection_bit;     /* 1 bit */
  unsigned bitrate_index;      /* 4 bits */
  unsigned sampling_frequency; /* 2 bits */
  unsigned padding_bit;        /* 1 bit */
  unsigned private_bit;        /* 1 bit */
  t_mpeg1_mode mode;           /* 2 bits */
  unsigned mode_extension;     /* 2 bits */
  unsigned copyright;          /* 1 bit */
  unsigned original_or_copy;   /* 1 bit */
  unsigned emphasis;           /* 2 bits */
}
t_mpeg1_header;
typedef struct {  /* MPEG1 Layer 3 Side Information : [2][2] means [gr][ch] */
  unsigned main_data_begin;         /* 9 bits */
  unsigned private_bits;            /* 3 bits in mono,5 in stereo */
  unsigned scfsi[2][4];             /* 1 bit */
  unsigned part2_3_length[2][2];    /* 12 bits */
  unsigned big_values[2][2];        /* 9 bits */
  unsigned global_gain[2][2];       /* 8 bits */
  unsigned scalefac_compress[2][2]; /* 4 bits */
  unsigned win_switch_flag[2][2];   /* 1 bit */
  /* if(win_switch_flag[][]) */ //use a union dammit
  unsigned block_type[2][2];        /* 2 bits */
  unsigned mixed_block_flag[2][2];  /* 1 bit */
  unsigned table_select[2][2][3];   /* 5 bits */
  unsigned subblock_gain[2][2][3];  /* 3 bits */
  /* else */
  /* table_select[][][] */
  unsigned region0_count[2][2];     /* 4 bits */
  unsigned region1_count[2][2];     /* 3 bits */
  /* end */
  unsigned preflag[2][2];           /* 1 bit */
  unsigned scalefac_scale[2][2];    /* 1 bit */
  unsigned count1table_select[2][2];/* 1 bit */
  unsigned count1[2][2];            /* Not in file,calc. by huff.dec.! */
}
t_mpeg1_side_info;
typedef struct { /* MPEG1 Layer 3 Main Data */
  unsigned  scalefac_l[2][2][21];    /* 0-4 bits */
  unsigned  scalefac_s[2][2][12][3]; /* 0-4 bits */
  float is[2][2][576];               /* Huffman coded freq. lines */
}
t_mpeg1_main_data;
typedef struct { /* Scale factor band indices,for long and short windows */
  unsigned l[23];
  unsigned s[14];
}
t_sf_band_indices;

#define C_SYNC             0xffe00000
#define C_EOF              0xffffffff
#define C_PI                   3.14159265358979323846
#define C_INV_SQRT_2           0.70710678118654752440
#define Hz                           1
#define kHz                    1000*Hz
#define bit_s                        1
#define kbit_s                 1000*bit_s
#define FRAG_SIZE_LN2     0x0011 /* 2^17=128kb */
#define FRAG_NUMS         0x0004

#define DBG(str,args...) { printf(str,## args); printf("\n"); }
#define ERR(str,args...) { fprintf(stderr,str,## args) ; fprintf(stderr,"\n"); }
#define EXIT(str,args...) { printf(str,## args);  printf("\n"); exit(0); }

#ifdef DEBUG //debug functions
static void dmp_fr(t_mpeg1_header *hdr);
static void dmp_si(t_mpeg1_header *hdr,t_mpeg1_side_info *si);
static void dmp_scf(t_mpeg1_side_info *si,t_mpeg1_main_data *md,int gr,int ch);
static void dmp_huff(t_mpeg1_main_data *md,int gr,int ch);
static void dmp_samples(t_mpeg1_main_data *md,int gr,int ch,int type);
#else
#define dmp_fr(...) do{}while(0)
#define dmp_si(...) do{}while(0)
#define dmp_scf(...) do{}while(0)
#define dmp_huff(...) do{}while(0)
#define dmp_samples(...) do{}while(0)
#endif

static int Read_Audio_L3(void);
static int Read_Header(void) ;
static int Read_Main_L3(void);

static void audio_write(unsigned *samples,unsigned nsamples,int sample_rate);
static void audio_write_raw(unsigned *samples,unsigned nsamples);
static void Decode_L3_Init_Song(void);
static void Error(const char *s,int e);
static void IMDCT_Win(float in[18],float out[36],unsigned block_type);
static void L3_Antialias(unsigned gr,unsigned ch);
static void L3_Frequency_Inversion(unsigned gr,unsigned ch);
static void L3_Hybrid_Synthesis(unsigned gr,unsigned ch);
static void L3_Requantize(unsigned gr,unsigned ch);
static void L3_Reorder(unsigned gr,unsigned ch);
static void L3_Stereo(unsigned gr);
static void L3_Subband_Synthesis(unsigned gr,unsigned ch,unsigned outdata[576]);
static void Read_Ancillary(void);
static void Read_Huffman(unsigned part_2_start,unsigned gr,unsigned ch);
static void Requantize_Process_Long(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb);
static void Requantize_Process_Short(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb,unsigned win);
static void Stereo_Process_Intensity_Long(unsigned gr,unsigned sfb);
static void Stereo_Process_Intensity_Short(unsigned gr,unsigned sfb);

static const unsigned g_mpeg1_bitrates[3 /* layer 1-3 */][15 /* header bitrate_index */] = {
  {    /* Layer 1 */
         0, 32000, 64000, 96000,128000,160000,192000,224000,
    256000,288000,320000,352000,384000,416000,448000
  },{  /* Layer 2 */
         0, 32000, 48000, 56000, 64000, 80000, 96000,112000,
    128000,160000,192000,224000,256000,320000,384000
  },{   /* Layer 3 */
         0, 32000, 40000, 48000, 56000, 64000, 80000, 96000,
    112000,128000,160000,192000,224000,256000,320000
  }
},
g_sampling_frequency[3] = { 44100 * Hz,48000 * Hz,32000 * Hz },
mpeg1_scalefac_sizes[16][2 /* slen1,slen2 */] = {
  {0,0},{0,1},{0,2},{0,3},{3,0},{1,1},{1,2},{1,3},
  {2,1},{2,2},{2,3},{3,1},{3,2},{3,3},{4,2},{4,3}
};

static const float ci[8]={-0.6,-0.535,-0.33,-0.185,-0.095,-0.041,-0.0142,-0.0037},
  cs[8]={0.857493,0.881742,0.949629,0.983315,0.995518,0.999161,0.999899,0.999993},
  ca[8]={-0.514496,-0.471732,-0.313377,-0.181913,-0.094574,-0.040966,-0.014199,-0.003700},
  is_ratios[6] = {0.000000f,0.267949f,0.577350f,1.000000f,1.732051f,3.732051f},
#ifdef IMDCT_TABLES
  g_imdct_win[4][36] = {
     {0.043619f,0.130526f,0.216440f,0.300706f,0.382683f,0.461749f,
      0.537300f,0.608761f,0.675590f,0.737277f,0.793353f,0.843391f,
      0.887011f,0.923880f,0.953717f,0.976296f,0.991445f,0.999048f,
      0.999048f,0.991445f,0.976296f,0.953717f,0.923879f,0.887011f,
      0.843391f,0.793353f,0.737277f,0.675590f,0.608761f,0.537299f,
      0.461748f,0.382683f,0.300706f,0.216439f,0.130526f,0.043619f
   },{0.043619f,0.130526f,0.216440f,0.300706f,0.382683f,0.461749f,
      0.537300f,0.608761f,0.675590f,0.737277f,0.793353f,0.843391f,
      0.887011f,0.923880f,0.953717f,0.976296f,0.991445f,0.999048f,
      1.000000f,1.000000f,1.000000f,1.000000f,1.000000f,1.000000f,
      0.991445f,0.923880f,0.793353f,0.608761f,0.382683f,0.130526f,
      0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f
   },{0.130526f,0.382683f,0.608761f,0.793353f,0.923880f,0.991445f,
      0.991445f,0.923880f,0.793353f,0.608761f,0.382683f,0.130526f,
      0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,
      0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,
      0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,
      0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,
   },{0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,0.000000f,
      0.130526f,0.382683f,0.608761f,0.793353f,0.923880f,0.991445f,
      1.000000f,1.000000f,1.000000f,1.000000f,1.000000f,1.000000f,
      0.999048f,0.991445f,0.976296f,0.953717f,0.923879f,0.887011f,
      0.843391f,0.793353f,0.737277f,0.675590f,0.608761f,0.537299f,
      0.461748f,0.382683f,0.300706f,0.216439f,0.130526f,0.043619f,
    }
  },
#endif
#ifdef IMDCT_NTABLES
  cos_N12[6][12] = {
     { 0.608761f,0.382683f,0.130526f,-0.130526f,-0.382683f,-0.608761f,
      -0.793353f,-0.923880f,-0.991445f,-0.991445f,-0.923879f,-0.793353f
   },{-0.923880f,-0.923879f,-0.382683f,0.382684f,0.923880f,0.923879f,
       0.382683f,-0.382684f,-0.923880f,-0.923879f,-0.382683f,0.382684f
   },{-0.130526f,0.923880f,0.608761f,-0.608762f,-0.923879f,0.130526f,
       0.991445f,0.382683f,-0.793354f,-0.793353f,0.382684f,0.991445f
   },{ 0.991445f,-0.382684f,-0.793353f,0.793354f,0.382683f,-0.991445f,
       0.130527f,0.923879f,-0.608762f,-0.608761f,0.923880f,0.130525f
   },{-0.382684f,-0.382683f,0.923879f,-0.923880f,0.382684f,0.382683f,
      -0.923879f,0.923880f,-0.382684f,-0.382683f,0.923879f,-0.923880f
   },{-0.793353f,0.923879f,-0.991445f,0.991445f,-0.923880f,0.793354f,
    -0.608762f,0.382684f,-0.130527f,-0.130525f,0.382682f,-0.608761f,},
  },
  cos_N36[18][36] = {
     { 0.675590f,0.608761f,0.537300f,0.461749f,0.382683f,0.300706f,
       0.216440f,0.130526f,0.043619f,-0.043619f,-0.130526f,-0.216440f,
      -0.300706f,-0.382684f,-0.461749f,-0.537300f,-0.608762f,-0.675590f,
      -0.737277f,-0.793353f,-0.843392f,-0.887011f,-0.923880f,-0.953717f,
      -0.976296f,-0.991445f,-0.999048f,-0.999048f,-0.991445f,-0.976296f,
      -0.953717f,-0.923879f,-0.887011f,-0.843391f,-0.793353f,-0.737277f
   },{-0.793353f,-0.923880f,-0.991445f,-0.991445f,-0.923879f,-0.793353f,
      -0.608761f,-0.382683f,-0.130526f,0.130526f,0.382684f,0.608762f,
       0.793354f,0.923880f,0.991445f,0.991445f,0.923879f,0.793353f,
       0.608761f,0.382683f,0.130526f,-0.130527f,-0.382684f,-0.608762f,
      -0.793354f,-0.923880f,-0.991445f,-0.991445f,-0.923879f,-0.793353f,
      -0.608761f,-0.382683f,-0.130526f,0.130527f,0.382684f,0.608762f
   },{-0.537299f,-0.130526f,0.300706f,0.675590f,0.923880f,0.999048f,
       0.887011f,0.608761f,0.216439f,-0.216440f,-0.608762f,-0.887011f,
      -0.999048f,-0.923879f,-0.675590f,-0.300705f,0.130527f,0.537300f,
       0.843392f,0.991445f,0.953717f,0.737277f,0.382683f,-0.043620f,
      -0.461749f,-0.793354f,-0.976296f,-0.976296f,-0.793353f,-0.461748f,
      -0.043618f,0.382684f,0.737278f,0.953717f,0.991445f,0.843391f
   },{ 0.887011f,0.991445f,0.737277f,0.216439f,-0.382684f,-0.843392f,
      -0.999048f,-0.793353f,-0.300705f,0.300706f,0.793354f,0.999048f,
       0.843391f,0.382683f,-0.216440f,-0.737278f,-0.991445f,-0.887010f,
      -0.461748f,0.130527f,0.675591f,0.976296f,0.923879f,0.537299f,
      -0.043621f,-0.608762f,-0.953717f,-0.953717f,-0.608760f,-0.043618f,
       0.537301f,0.923880f,0.976296f,0.675589f,0.130525f,-0.461750f
   },{ 0.382683f,-0.382684f,-0.923880f,-0.923879f,-0.382683f,0.382684f,
       0.923880f,0.923879f,0.382683f,-0.382684f,-0.923880f,-0.923879f,
      -0.382683f,0.382684f,0.923880f,0.923879f,0.382682f,-0.382685f,
      -0.923880f,-0.923879f,-0.382682f,0.382685f,0.923880f,0.923879f,
       0.382682f,-0.382685f,-0.923880f,-0.923879f,-0.382682f,0.382685f,
       0.923880f,0.923879f,0.382682f,-0.382685f,-0.923880f,-0.923879f
   },{-0.953717f,-0.793353f,0.043620f,0.843392f,0.923879f,0.216439f,
      -0.675591f,-0.991445f,-0.461748f,0.461749f,0.991445f,0.675589f,
      -0.216441f,-0.923880f,-0.843391f,-0.043618f,0.793354f,0.953717f,
       0.300704f,-0.608763f,-0.999048f,-0.537298f,0.382685f,0.976296f,
       0.737276f,-0.130528f,-0.887012f,-0.887010f,-0.130524f,0.737279f,
       0.976296f,0.382681f,-0.537301f,-0.999048f,-0.608760f,0.300708f
   },{-0.216439f,0.793354f,0.887010f,-0.043620f,-0.923880f,-0.737277f,
       0.300707f,0.991445f,0.537299f,-0.537301f,-0.991445f,-0.300705f,
       0.737278f,0.923879f,0.043618f,-0.887012f,-0.793352f,0.216441f,
       0.976296f,0.608760f,-0.461750f,-0.999048f,-0.382682f,0.675592f,
       0.953716f,0.130524f,-0.843393f,-0.843390f,0.130529f,0.953718f,
       0.675588f,-0.382686f,-0.999048f,-0.461746f,0.608764f,0.976295f
   },{ 0.991445f,0.382683f,-0.793354f,-0.793353f,0.382684f,0.991445f,
       0.130525f,-0.923880f,-0.608760f,0.608763f,0.923879f,-0.130528f,
      -0.991445f,-0.382682f,0.793354f,0.793352f,-0.382685f,-0.991445f,
      -0.130524f,0.923880f,0.608760f,-0.608763f,-0.923879f,0.130529f,
       0.991445f,0.382681f,-0.793355f,-0.793352f,0.382686f,0.991444f,
       0.130523f,-0.923881f,-0.608759f,0.608764f,0.923878f,-0.130529f
   },{ 0.043619f,-0.991445f,-0.216439f,0.953717f,0.382682f,-0.887011f,
      -0.537299f,0.793354f,0.675589f,-0.675591f,-0.793352f,0.537301f,
       0.887010f,-0.382685f,-0.953716f,0.216442f,0.991445f,-0.043622f,
      -0.999048f,-0.130524f,0.976297f,0.300703f,-0.923881f,-0.461746f,
       0.843393f,0.608759f,-0.737279f,-0.737275f,0.608764f,0.843390f,
      -0.461752f,-0.923878f,0.300709f,0.976295f,-0.130530f,-0.999048f
   },{-0.999048f,0.130527f,0.976296f,-0.300707f,-0.923879f,0.461750f,
       0.843391f,-0.608763f,-0.737276f,0.737279f,0.608760f,-0.843392f,
      -0.461747f,0.923880f,0.300704f,-0.976297f,-0.130524f,0.999048f,
      -0.043622f,-0.991445f,0.216442f,0.953716f,-0.382686f,-0.887009f,
       0.537302f,0.793351f,-0.675593f,-0.675588f,0.793355f,0.537297f,
      -0.887013f,-0.382680f,0.953718f,0.216436f,-0.991445f,-0.043615f
   },{ 0.130527f,0.923879f,-0.608762f,-0.608760f,0.923880f,0.130525f,
      -0.991445f,0.382685f,0.793352f,-0.793355f,-0.382682f,0.991445f,
      -0.130528f,-0.923879f,0.608763f,0.608759f,-0.923881f,-0.130523f,
       0.991444f,-0.382686f,-0.793351f,0.793355f,0.382680f,-0.991445f,
       0.130530f,0.923878f,-0.608764f,-0.608758f,0.923881f,0.130522f,
      -0.991444f,0.382687f,0.793351f,-0.793356f,-0.382679f,0.991445f
   },{ 0.976296f,-0.608762f,-0.461747f,0.999048f,-0.382685f,-0.675589f,
       0.953717f,-0.130528f,-0.843390f,0.843393f,0.130524f,-0.953716f,
       0.675592f,0.382681f,-0.999048f,0.461751f,0.608759f,-0.976297f,
       0.216443f,0.793351f,-0.887012f,-0.043616f,0.923878f,-0.737280f,
      -0.300702f,0.991444f,-0.537303f,-0.537296f,0.991445f,-0.300710f,
      -0.737274f,0.923881f,-0.043624f,-0.887009f,0.793356f,0.216435f
   },{-0.300707f,-0.608760f,0.999048f,-0.537301f,-0.382682f,0.976296f,
      -0.737279f,-0.130524f,0.887010f,-0.887012f,0.130529f,0.737276f,
      -0.976297f,0.382686f,0.537297f,-0.999048f,0.608764f,0.300703f,
      -0.953716f,0.793355f,0.043616f,-0.843389f,0.923881f,-0.216444f,
      -0.675587f,0.991445f,-0.461752f,-0.461745f,0.991444f,-0.675594f,
      -0.216435f,0.923878f,-0.843394f,0.043625f,0.793350f,-0.953719f
   },{-0.923879f,0.923880f,-0.382685f,-0.382682f,0.923879f,-0.923880f,
       0.382685f,0.382681f,-0.923879f,0.923880f,-0.382686f,-0.382681f,
       0.923878f,-0.923881f,0.382686f,0.382680f,-0.923878f,0.923881f,
      -0.382687f,-0.382680f,0.923878f,-0.923881f,0.382687f,0.382679f,
      -0.923878f,0.923881f,-0.382688f,-0.382679f,0.923878f,-0.923881f,
       0.382688f,0.382678f,-0.923877f,0.923882f,-0.382689f,-0.382678f
   },{ 0.461750f,0.130525f,-0.675589f,0.976296f,-0.923880f,0.537301f,
       0.043617f,-0.608760f,0.953716f,-0.953718f,0.608764f,-0.043622f,
      -0.537297f,0.923878f,-0.976297f,0.675593f,-0.130530f,-0.461745f,
       0.887009f,-0.991445f,0.737280f,-0.216444f,-0.382679f,0.843389f,
      -0.999048f,0.793356f,-0.300711f,-0.300701f,0.793350f,-0.999048f,
       0.843394f,-0.382689f,-0.216434f,0.737273f,-0.991444f,0.887014f
   },{ 0.843391f,-0.991445f,0.953717f,-0.737279f,0.382685f,0.043617f,
      -0.461747f,0.793352f,-0.976295f,0.976297f,-0.793355f,0.461751f,
      -0.043623f,-0.382680f,0.737275f,-0.953716f,0.991445f,-0.843394f,
       0.537303f,-0.130530f,-0.300702f,0.675587f,-0.923878f,0.999048f,
      -0.887013f,0.608766f,-0.216445f,-0.216434f,0.608757f,-0.887008f,
       0.999048f,-0.923882f,0.675595f,-0.300712f,-0.130520f,0.537294f
   },{-0.608763f,0.382685f,-0.130528f,-0.130524f,0.382681f,-0.608760f,
       0.793352f,-0.923879f,0.991444f,-0.991445f,0.923881f,-0.793355f,
       0.608764f,-0.382687f,0.130530f,0.130522f,-0.382680f,0.608758f,
      -0.793351f,0.923878f,-0.991444f,0.991446f,-0.923881f,0.793357f,
      -0.608766f,0.382689f,-0.130532f,-0.130520f,0.382678f,-0.608756f,
       0.793349f,-0.923877f,0.991444f,-0.991446f,0.923882f,-0.793358f
   },{-0.737276f,0.793352f,-0.843390f,0.887010f,-0.923879f,0.953716f,
      -0.976295f,0.991444f,-0.999048f,0.999048f,-0.991445f,0.976297f,
      -0.953718f,0.923881f,-0.887013f,0.843394f,-0.793356f,0.737280f,
      -0.675594f,0.608765f,-0.537304f,0.461753f,-0.382688f,0.300711f,
      -0.216445f,0.130532f,-0.043625f,-0.043613f,0.130520f,-0.216433f,
       0.300699f,-0.382677f,0.461742f,-0.537293f,0.608755f,-0.675585f
   }};
#endif
#ifdef POW34_ITERATE
  static const float powtab34[32] = {
  0.000000f,1.000000f,2.519842f,4.326749f,6.349605f,8.549880f,10.902724f,
  13.390519f,16.000001f,18.720756f,21.544349f,24.463783f,27.473145f,30.567354f,
  33.741995f,36.993185f,40.317478f,43.711792f,47.173351f,50.699637f,54.288359f,
  57.937415f,61.644873f,65.408949f,69.227988f,73.100453f,77.024908f,81.000011f,
  85.024502f,89.097200f,93.216988f,97.382814f
},
#endif
  g_synth_dtbl[512] = {
   0.000000000,-0.000015259,-0.000015259,-0.000015259,
  -0.000015259,-0.000015259,-0.000015259,-0.000030518,
  -0.000030518,-0.000030518,-0.000030518,-0.000045776,
  -0.000045776,-0.000061035,-0.000061035,-0.000076294,
  -0.000076294,-0.000091553,-0.000106812,-0.000106812,
  -0.000122070,-0.000137329,-0.000152588,-0.000167847,
  -0.000198364,-0.000213623,-0.000244141,-0.000259399,
  -0.000289917,-0.000320435,-0.000366211,-0.000396729,
  -0.000442505,-0.000473022,-0.000534058,-0.000579834,
  -0.000625610,-0.000686646,-0.000747681,-0.000808716,
  -0.000885010,-0.000961304,-0.001037598,-0.001113892,
  -0.001205444,-0.001296997,-0.001388550,-0.001480103,
  -0.001586914,-0.001693726,-0.001785278,-0.001907349,
  -0.002014160,-0.002120972,-0.002243042,-0.002349854,
  -0.002456665,-0.002578735,-0.002685547,-0.002792358,
  -0.002899170,-0.002990723,-0.003082275,-0.003173828,
   0.003250122, 0.003326416, 0.003387451, 0.003433228,
   0.003463745, 0.003479004, 0.003479004, 0.003463745,
   0.003417969, 0.003372192, 0.003280640, 0.003173828,
   0.003051758, 0.002883911, 0.002700806, 0.002487183,
   0.002227783, 0.001937866, 0.001617432, 0.001266479,
   0.000869751, 0.000442505,-0.000030518,-0.000549316,
  -0.001098633,-0.001693726,-0.002334595,-0.003005981,
  -0.003723145,-0.004486084,-0.005294800,-0.006118774,
  -0.007003784,-0.007919312,-0.008865356,-0.009841919,
  -0.010848999,-0.011886597,-0.012939453,-0.014022827,
  -0.015121460,-0.016235352,-0.017349243,-0.018463135,
  -0.019577026,-0.020690918,-0.021789551,-0.022857666,
  -0.023910522,-0.024932861,-0.025909424,-0.026840210,
  -0.027725220,-0.028533936,-0.029281616,-0.029937744,
  -0.030532837,-0.031005859,-0.031387329,-0.031661987,
  -0.031814575,-0.031845093,-0.031738281,-0.031478882,
   0.031082153, 0.030517578, 0.029785156, 0.028884888,
   0.027801514, 0.026535034, 0.025085449, 0.023422241,
   0.021575928, 0.019531250, 0.017257690, 0.014801025,
   0.012115479, 0.009231567, 0.006134033, 0.002822876,
  -0.000686646,-0.004394531,-0.008316040,-0.012420654,
  -0.016708374,-0.021179199,-0.025817871,-0.030609131,
  -0.035552979,-0.040634155,-0.045837402,-0.051132202,
  -0.056533813,-0.061996460,-0.067520142,-0.073059082,
  -0.078628540,-0.084182739,-0.089706421,-0.095169067,
  -0.100540161,-0.105819702,-0.110946655,-0.115921021,
  -0.120697021,-0.125259399,-0.129562378,-0.133590698,
  -0.137298584,-0.140670776,-0.143676758,-0.146255493,
  -0.148422241,-0.150115967,-0.151306152,-0.151962280,
  -0.152069092,-0.151596069,-0.150497437,-0.148773193,
  -0.146362305,-0.143264771,-0.139450073,-0.134887695,
  -0.129577637,-0.123474121,-0.116577148,-0.108856201,
   0.100311279, 0.090927124, 0.080688477, 0.069595337,
   0.057617188, 0.044784546, 0.031082153, 0.016510010,
   0.001068115,-0.015228271,-0.032379150,-0.050354004,
  -0.069168091,-0.088775635,-0.109161377,-0.130310059,
  -0.152206421,-0.174789429,-0.198059082,-0.221984863,
  -0.246505737,-0.271591187,-0.297210693,-0.323318481,
  -0.349868774,-0.376800537,-0.404083252,-0.431655884,
  -0.459472656,-0.487472534,-0.515609741,-0.543823242,
  -0.572036743,-0.600219727,-0.628295898,-0.656219482,
  -0.683914185,-0.711318970,-0.738372803,-0.765029907,
  -0.791213989,-0.816864014,-0.841949463,-0.866363525,
  -0.890090942,-0.913055420,-0.935195923,-0.956481934,
  -0.976852417,-0.996246338,-1.014617920,-1.031936646,
  -1.048156738,-1.063217163,-1.077117920,-1.089782715,
  -1.101211548,-1.111373901,-1.120223999,-1.127746582,
  -1.133926392,-1.138763428,-1.142211914,-1.144287109,
   1.144989014, 1.144287109, 1.142211914, 1.138763428,
   1.133926392, 1.127746582, 1.120223999, 1.111373901,
   1.101211548, 1.089782715, 1.077117920, 1.063217163,
   1.048156738, 1.031936646, 1.014617920, 0.996246338,
   0.976852417, 0.956481934, 0.935195923, 0.913055420,
   0.890090942, 0.866363525, 0.841949463, 0.816864014,
   0.791213989, 0.765029907, 0.738372803, 0.711318970,
   0.683914185, 0.656219482, 0.628295898, 0.600219727,
   0.572036743, 0.543823242, 0.515609741, 0.487472534,
   0.459472656, 0.431655884, 0.404083252, 0.376800537,
   0.349868774, 0.323318481, 0.297210693, 0.271591187,
   0.246505737, 0.221984863, 0.198059082, 0.174789429,
   0.152206421, 0.130310059, 0.109161377, 0.088775635,
   0.069168091, 0.050354004, 0.032379150, 0.015228271,
  -0.001068115,-0.016510010,-0.031082153,-0.044784546,
  -0.057617188,-0.069595337,-0.080688477,-0.090927124,
   0.100311279, 0.108856201, 0.116577148, 0.123474121,
   0.129577637, 0.134887695, 0.139450073, 0.143264771,
   0.146362305, 0.148773193, 0.150497437, 0.151596069,
   0.152069092, 0.151962280, 0.151306152, 0.150115967,
   0.148422241, 0.146255493, 0.143676758, 0.140670776,
   0.137298584, 0.133590698, 0.129562378, 0.125259399,
   0.120697021, 0.115921021, 0.110946655, 0.105819702,
   0.100540161, 0.095169067, 0.089706421, 0.084182739,
   0.078628540, 0.073059082, 0.067520142, 0.061996460,
   0.056533813, 0.051132202, 0.045837402, 0.040634155,
   0.035552979, 0.030609131, 0.025817871, 0.021179199,
   0.016708374, 0.012420654, 0.008316040, 0.004394531,
   0.000686646,-0.002822876,-0.006134033,-0.009231567,
  -0.012115479,-0.014801025,-0.017257690,-0.019531250,
  -0.021575928,-0.023422241,-0.025085449,-0.026535034,
  -0.027801514,-0.028884888,-0.029785156,-0.030517578,
   0.031082153, 0.031478882, 0.031738281, 0.031845093,
   0.031814575, 0.031661987, 0.031387329, 0.031005859,
   0.030532837, 0.029937744, 0.029281616, 0.028533936,
   0.027725220, 0.026840210, 0.025909424, 0.024932861,
   0.023910522, 0.022857666, 0.021789551, 0.020690918,
   0.019577026, 0.018463135, 0.017349243, 0.016235352,
   0.015121460, 0.014022827, 0.012939453, 0.011886597,
   0.010848999, 0.009841919, 0.008865356, 0.007919312,
   0.007003784, 0.006118774, 0.005294800, 0.004486084,
   0.003723145, 0.003005981, 0.002334595, 0.001693726,
   0.001098633, 0.000549316, 0.000030518,-0.000442505,
  -0.000869751,-0.001266479,-0.001617432,-0.001937866,
  -0.002227783,-0.002487183,-0.002700806,-0.002883911,
  -0.003051758,-0.003173828,-0.003280640,-0.003372192,
  -0.003417969,-0.003463745,-0.003479004,-0.003479004,
  -0.003463745,-0.003433228,-0.003387451,-0.003326416,
   0.003250122, 0.003173828, 0.003082275, 0.002990723,
   0.002899170, 0.002792358, 0.002685547, 0.002578735,
   0.002456665, 0.002349854, 0.002243042, 0.002120972,
   0.002014160, 0.001907349, 0.001785278, 0.001693726,
   0.001586914, 0.001480103, 0.001388550, 0.001296997,
   0.001205444, 0.001113892, 0.001037598, 0.000961304,
   0.000885010, 0.000808716, 0.000747681, 0.000686646,
   0.000625610, 0.000579834, 0.000534058, 0.000473022,
   0.000442505, 0.000396729, 0.000366211, 0.000320435,
   0.000289917, 0.000259399, 0.000244141, 0.000213623,
   0.000198364, 0.000167847, 0.000152588, 0.000137329,
   0.000122070, 0.000106812, 0.000106812, 0.000091553,
   0.000076294, 0.000076294, 0.000061035, 0.000061035,
   0.000045776, 0.000045776, 0.000030518, 0.000030518,
   0.000030518, 0.000030518, 0.000015259, 0.000015259,
   0.000015259, 0.000015259, 0.000015259, 0.000015259,
//},g_synth_n_win[64][32]={
};


static unsigned hsynth_init = 1,synth_init = 1;

/* Scale factor band indices
 *
 * One table per sample rate. Each table contains the frequency indices
 * for the 12 short and 21 long scalefactor bands. The short indices
 * must be multiplied by 3 to get the actual index.
 */
static const t_sf_band_indices g_sf_band_indices[3 /* Sampling freq. */] = {
    {
      {0,4,8,12,16,20,24,30,36,44,52,62,74,90,110,134,162,196,238,288,342,418,576},
      {0,4,8,12,16,22,30,40,52,66,84,106,136,192}
    },
    {
      {0,4,8,12,16,20,24,30,36,42,50,60,72,88,106,128,156,190,230,276,330,384,576},
      {0,4,8,12,16,22,28,38,50,64,80,100,126,192}
    },
    {
      {0,4,8,12,16,20,24,30,36,44,54,66,82,102,126,156,194,240,296,364,448,550,576},
      {0,4,8,12,16,22,30,42,58,78,104,138,180,192}
    }
  };

static t_mpeg1_header    g_frame_header;
static t_mpeg1_side_info g_side_info;  /* < 100 words */
static t_mpeg1_main_data g_main_data;  /* Large static data(~2500 words) */

#ifdef DEBUG
static void dmp_fr(t_mpeg1_header *hdr){
  printf("rate %d,sfreq %d,pad %d,mod %d,modext %d,emph %d\n",
          hdr->bitrate_index,hdr->sampling_frequency,hdr->padding_bit,
          hdr->mode,hdr->mode_extension,hdr->emphasis);
}

static void dmp_si(t_mpeg1_header *hdr,t_mpeg1_side_info *si){
  int nch,ch,gr;

  nch = hdr->mode == mpeg1_mode_single_channel ? 1 : 2;
  printf("main_data_begin %d,priv_bits %d\n",si->main_data_begin,si->private_bits);
  for(ch = 0; ch < nch; ch++) {
    printf("scfsi %d %d %d %d\n",si->scfsi[ch][0],si->scfsi[ch][1],si->scfsi[ch][2],si->scfsi[ch][3]);
    for(gr = 0; gr < 2; gr++) {
      printf("p23l %d,bv %d,gg %d,scfc %d,wsf %d,bt %d\n",
              si->part2_3_length[gr][ch],si->big_values[gr][ch],
              si->global_gain[gr][ch],si->scalefac_compress[gr][ch],
              si->win_switch_flag[gr][ch],si->block_type[gr][ch]);
      if(si->win_switch_flag[gr][ch]) {
        printf("mbf %d,ts1 %d,ts2 %d,sbg1 %d,sbg2 %d,sbg3 %d\n",
                si->mixed_block_flag[gr][ch],si->table_select[gr][ch][0],
                si->table_select[gr][ch][1],si->subblock_gain[gr][ch][0],
                si->subblock_gain[gr][ch][1],si->subblock_gain[gr][ch][2]);
      }else{
        printf("ts1 %d,ts2 %d,ts3 %d\n",si->table_select[gr][ch][0],
                si->table_select[gr][ch][1],si->table_select[gr][ch][2]);
      }
      printf("r0c %d,r1c %d\n",si->region0_count[gr][ch],si->region1_count[gr][ch]);
      printf("pf %d,scfs %d,c1ts %d\n",si->preflag[gr][ch],si->scalefac_scale[gr][ch],si->count1table_select[gr][ch]);
    }
  }
}

static void dmp_scf(t_mpeg1_side_info *si,t_mpeg1_main_data *md,int gr,int ch){
  int sfb,win;

  if((si->win_switch_flag[gr][ch] != 0) &&(si->block_type[gr][ch] == 2)) {
    if(si->mixed_block_flag[gr][ch] != 0) { /* First the long block scalefacs */
      for(sfb = 0; sfb < 8; sfb++)
        printf("scfl%d %d%s",sfb,md->scalefac_l[gr][ch][sfb],(sfb==7)?"\n":",");
      for(sfb = 3; sfb < 12; sfb++) /* And next the short block scalefacs */
        for(win = 0; win < 3; win++)
          printf("scfs%d,%d %d%s",sfb,win,md->scalefac_s[gr][ch][sfb][win],(win==2)?"\n":",");
    }else{                /* Just short blocks */
      for(sfb = 0; sfb < 12; sfb++)
        for(win = 0; win < 3; win++)
          printf("scfs%d,%d %d%s",sfb,win,md->scalefac_s[gr][ch][sfb][win],(win==2)?"\n":",");
    }
  }else for(sfb = 0; sfb < 21; sfb++) /* Just long blocks; scalefacs first */
          printf("scfl%d %d%s",sfb,md->scalefac_l[gr][ch][sfb], (sfb == 20)?"\n":",");
}

static void dmp_huff(t_mpeg1_main_data *md,int gr,int ch){
  int i;
  printf("HUFFMAN\n");
  for(i = 0; i < 576; i++) printf("%d: %d\n",i,(int) md->is[gr][ch][i]);
}

static void dmp_samples(t_mpeg1_main_data *md,int gr,int ch,int type){
  int i,val;
  extern double rint(double);

  printf("SAMPLES%d\n",type);
  for(i = 0; i < 576; i++) {
    val =(int) rint(md->is[gr][ch][i] * 32768.0);
    if(val >= 32768) val = 32767;
    if(val < -32768) val = -32768;
    printf("%d: %d\n",i,val);
  }
}
#endif

/**Description: calculates y=x^(4/3) when requantizing samples.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static inline float Requantize_Pow_43(unsigned is_pos){
#ifdef POW34_TABLE
  static float powtab34[8207];
  static int init = 0;
  int i;

  if(init == 0) {   /* First time initialization */
    for(i = 0; i < 8207; i++)
      powtab34[i] = pow((float) i,4.0 / 3.0);
    init = 1;
  }
#ifdef DEBUG
  if(is_pos > 8206) {
    ERR("is_pos = %d larger than 8206!",is_pos);
    is_pos = 8206;
  }
#endif /* DEBUG */
  return(powtab34[is_pos]);  /* Done */
#elif defined POW34_ITERATE
  float a4,a2,x,x2,x3,x_next,is_f1,is_f2,is_f3;
  unsigned i;
//static unsigned init = 0;
//static float powtab34[32];
  static float coeff[3] = {-1.030797119e+02,6.319399834e+00,2.395095071e-03};
//if(init == 0) { /* First time initialization */
//  for(i = 0; i < 32; i++) powtab34[i] = pow((float) i,4.0 / 3.0);
//  init = 1;
//}
  /* We use a table for 0<is_pos<32 since they are so common */
  if(is_pos < 32) return(powtab34[is_pos]);
  a2 = is_pos * is_pos;
  a4 = a2 * a2;
  is_f1 =(float) is_pos;
  is_f2 = is_f1 * is_f1;
  is_f3 = is_f1 * is_f2;
  /*  x = coeff[0] + coeff[1]*is_f1 + coeff[2]*is_f2 + coeff[3]*is_f3; */
  x = coeff[0] + coeff[1]*is_f1 + coeff[2]*is_f2;
  for(i = 0; i < 3; i++) {
    x2 = x*x;
    x3 = x*x2;
    x_next =(2*x3 + a4) /(3*x2);
    x = x_next;
  }
  return(x);
#else /* no optimization */
return powf((float)is_pos,4.0f / 3.0f);
#endif /* POW34_TABLE || POW34_ITERATE */
}

/**Description: decodes a layer 3 bitstream into audio samples.
* Parameters: Outdata vector.
* Return value: OK or ERROR if the frame contains errors.
* Author: Krister Lagerström(krister@kmlager.com) **/
int Decode_L3(void){
  unsigned gr,ch,nch,out[576];

  /* Number of channels(1 for mono and 2 for stereo) */
  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);
  for(gr = 0; gr < 2; gr++) {
    for(ch = 0; ch < nch; ch++) {
      dmp_scf(&g_side_info,&g_main_data,gr,ch); //noop unless debug
      dmp_huff(&g_main_data,gr,ch); //noop unless debug
      L3_Requantize(gr,ch); /* Requantize samples */
      dmp_samples(&g_main_data,gr,ch,0); //noop unless debug
      L3_Reorder(gr,ch); /* Reorder short blocks */
    } /* end for(ch... */
    L3_Stereo(gr); /* Stereo processing */
    dmp_samples(&g_main_data,gr,0,1); //noop unless debug
    dmp_samples(&g_main_data,gr,1,1); //noop unless debug
    for(ch = 0; ch < nch; ch++) {
      L3_Antialias(gr,ch); /* Antialias */
      dmp_samples(&g_main_data,gr,ch,2); //noop unless debug
      L3_Hybrid_Synthesis(gr,ch); /*(IMDCT,windowing,overlapp add) */
      L3_Frequency_Inversion(gr,ch); /* Frequency inversion */
     dmp_samples(&g_main_data,gr,ch,3); //noop unless debug
      L3_Subband_Synthesis(gr,ch,out); /* Polyphase subband synthesis */
    } /* end for(ch... */
#ifdef DEBUG
    {
      int i,ctr = 0;
      printf("PCM:\n");
      for(i = 0; i < 576; i++) {
        printf("%d: %d\n",ctr++,(out[i] >> 16) & 0xffff);
        if(nch == 2) printf("%d: %d\n",ctr++,out[i] & 0xffff);
      }
    }
#endif /* DEBUG */
     /*FIXME - replace with simple interface stream*/
    audio_write((unsigned *) out,576,
                 g_sampling_frequency[g_frame_header.sampling_frequency]);
  } /* end for(gr... */
  return(OK);   /* Done */
}

/**Description: Reads audio and main data from bitstream into a buffer. main
*  data is taken from this frame and up to 2 previous frames.
* Parameters: None
* Return value: OK or ERROR if data could not be read,or contains errors.
* Author: Krister Lagerström(krister@kmlager.com) **/
static int Read_Audio_L3(void){
  unsigned framesize,sideinfo_size,main_data_size,nch,ch,gr,scfsi_band,region,window;

  /* Number of channels(1 for mono and 2 for stereo) */
  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);
  /* Calculate header audio data size */
  framesize = (144 *
    g_mpeg1_bitrates[g_frame_header.layer-1][g_frame_header.bitrate_index]) /
    g_sampling_frequency[g_frame_header.sampling_frequency] +
    g_frame_header.padding_bit;
  if(framesize > 2000) {
    ERR("framesize = %d\n",framesize);
    return(ERROR);
  }
  /* Sideinfo is 17 bytes for one channel and 32 bytes for two */
  sideinfo_size =(nch == 1 ? 17 : 32);
  /* Main data size is the rest of the frame,including ancillary data */
  main_data_size = framesize - sideinfo_size - 4 /* sync+header */;
  /* CRC is 2 bytes */
  if(g_frame_header.protection_bit == 0) main_data_size -= 2;
  /* DBG("framesize      =   %d\n",framesize); */
  /* DBG("sideinfo_size  =   %d\n",sideinfo_size); */
  /* DBG("main_data_size =   %d\n",main_data_size); */
  /* Read sideinfo from bitstream into buffer used by Get_Side_Bits() */
  Get_Sideinfo(sideinfo_size);
  if(Get_Filepos() == C_EOF) return(ERROR);
  /* Parse audio data */
  /* Pointer to where we should start reading main data */
  g_side_info.main_data_begin = Get_Side_Bits(9);
  /* Get private bits. Not used for anything. */
  if(g_frame_header.mode == mpeg1_mode_single_channel)
    g_side_info.private_bits = Get_Side_Bits(5);
  else g_side_info.private_bits = Get_Side_Bits(3);
  /* Get scale factor selection information */
  for(ch = 0; ch < nch; ch++)
    for(scfsi_band = 0; scfsi_band < 4; scfsi_band++)
      g_side_info.scfsi[ch][scfsi_band] = Get_Side_Bits(1);
  /* Get the rest of the side information */
  for(gr = 0; gr < 2; gr++) {
    for(ch = 0; ch < nch; ch++) {
      g_side_info.part2_3_length[gr][ch]    = Get_Side_Bits(12);
      g_side_info.big_values[gr][ch]        = Get_Side_Bits(9);
      g_side_info.global_gain[gr][ch]       = Get_Side_Bits(8);
      g_side_info.scalefac_compress[gr][ch] = Get_Side_Bits(4);
      g_side_info.win_switch_flag[gr][ch]   = Get_Side_Bits(1);
      if(g_side_info.win_switch_flag[gr][ch] == 1) {
        g_side_info.block_type[gr][ch]       = Get_Side_Bits(2);
        g_side_info.mixed_block_flag[gr][ch] = Get_Side_Bits(1);
        for(region = 0; region < 2; region++)
          g_side_info.table_select[gr][ch][region] = Get_Side_Bits(5);
        for(window = 0; window < 3; window++)
          g_side_info.subblock_gain[gr][ch][window] = Get_Side_Bits(3);
        if((g_side_info.block_type[gr][ch]==2)&&(g_side_info.mixed_block_flag[gr][ch]==0))
          g_side_info.region0_count[gr][ch] = 8; /* Implicit */
        else g_side_info.region0_count[gr][ch] = 7; /* Implicit */
        /* The standard is wrong on this!!! */   /* Implicit */
        g_side_info.region1_count[gr][ch] = 20 - g_side_info.region0_count[gr][ch];
     }else{
       for(region = 0; region < 3; region++)
         g_side_info.table_select[gr][ch][region] = Get_Side_Bits(5);
       g_side_info.region0_count[gr][ch] = Get_Side_Bits(4);
       g_side_info.region1_count[gr][ch] = Get_Side_Bits(3);
       g_side_info.block_type[gr][ch] = 0;  /* Implicit */
      }  /* end if ... */
      g_side_info.preflag[gr][ch]            = Get_Side_Bits(1);
      g_side_info.scalefac_scale[gr][ch]     = Get_Side_Bits(1);
      g_side_info.count1table_select[gr][ch] = Get_Side_Bits(1);
    } /* end for(channel... */
  } /* end for(granule... */
  return(OK);/* Done */

}

/**Description: Search for next frame and read it into  buffer. Main data in
   this frame is saved for two frames since it might be needed by them.
* Parameters: None
* Return value: OK if a frame is successfully read,ERROR otherwise.
* Author: Krister Lagerström(krister@kmlager.com) **/
int Read_Frame(void){
  unsigned first = 0;

  if(Get_Filepos() == 0) Decode_L3_Init_Song();
  /* Try to find the next frame in the bitstream and decode it */
  if(Read_Header() != OK) return(ERROR);
#ifdef DEBUG
  { static int framenum = 0;
    printf("\nFrame %d\n",framenum++);
    dmp_fr(&g_frame_header);
  }
    DBG("Starting decode,Layer: %d,Rate: %6d,Sfreq: %05d",
         g_frame_header.layer,
         g_mpeg1_bitrates[g_frame_header.layer - 1][g_frame_header.bitrate_index],
         g_sampling_frequency[g_frame_header.sampling_frequency]);
#endif
  /* Get CRC word if present */
  if((g_frame_header.protection_bit==0)&&(Read_CRC()!=OK)) return(ERROR);
  if(g_frame_header.layer == 3) {  /* Get audio data */
    Read_Audio_L3();  /* Get side info */
    dmp_si(&g_frame_header,&g_side_info); /* DEBUG */
    /* If there's not enough main data in the bit reservoir,
     * signal to calling function so that decoding isn't done! */
    /* Get main data(scalefactors and Huffman coded frequency data) */
    if(Read_Main_L3() != OK) return(ERROR);
  }else{
    ERR("Only layer 3(!= %d) is supported!\n",g_frame_header.layer);
    return(ERROR);
  }
  return(OK);
}

static bool is_header(unsigned header) {
  /* Are the high 11 bits the syncword(0xffe)? */
  if ((header & C_SYNC) != C_SYNC) {
    return false;
  }
  // Bitrate must not be 15.
  if ((header & (0xf<<12)) == 0xf<<12) {
    return false;
  }
  // Sample Frequency must not be 3.
  if ((header & (3<<10)) == 3<<10) {
    return false;
  }
  return true;
}

/**Description: Scans bitstream for syncword until we find it or EOF. The
   syncword must be byte-aligned. It then reads and parses audio header.
* Parameters: None
* Return value: OK or ERROR if the syncword can't be found,or the header
*               contains impossible values.
* Author: Krister Lagerström(krister@kmlager.com) **/
static int Read_Header(void) {
  unsigned b1,b2,b3,b4,header;

  /* Get the next four bytes from the bitstream */
  b1 = Get_Byte();
  b2 = Get_Byte();
  b3 = Get_Byte();
  b4 = Get_Byte();
  /* If we got an End Of File condition we're done */
  if((b1==C_EOF)||(b2==C_EOF)||(b3==C_EOF)||(b4==C_EOF))
    return(ERROR);
  header =(b1 << 24) |(b2 << 16) |(b3 << 8) |(b4 << 0);
  while(!is_header(header)) {
    /* No,so scan the bitstream one byte at a time until we find it or EOF */
    /* Shift the values one byte to the left */
    b1 = b2;
    b2 = b3;
    b3 = b4;
    /* Get one new byte from the bitstream */
    b4 = Get_Byte();
    /* If we got an End Of File condition we're done */
    if(b4 == C_EOF) return(ERROR);
    /* Make up the new header */
    header = (b1 << 24) | (b2 << 16) | (b3 << 8) | (b4 << 0);
  } /* while... */
  /* If we get here we've found the sync word,and can decode the header
   * which is in the low 20 bits of the 32-bit sync+header word. */
  /* Decode the header */
  g_frame_header.id                 =(header & 0x00180000) >> 19;
  g_frame_header.layer              =(header & 0x00060000) >> 17;
  g_frame_header.protection_bit     =(header & 0x00010000) >> 16;
  g_frame_header.bitrate_index      =(header & 0x0000f000) >> 12;
  g_frame_header.sampling_frequency =(header & 0x00000c00) >> 10;
  g_frame_header.padding_bit        =(header & 0x00000200) >> 9;
  g_frame_header.private_bit        =(header & 0x00000100) >> 8;
  g_frame_header.mode               =(header & 0x000000c0) >> 6;
  g_frame_header.mode_extension     =(header & 0x00000030) >> 4;
  g_frame_header.copyright          =(header & 0x00000008) >> 3;
  g_frame_header.original_or_copy   =(header & 0x00000004) >> 2;
  g_frame_header.emphasis           =(header & 0x00000003) >> 0;
  /* Check for invalid values and impossible combinations */
  if(g_frame_header.id != 3) {
    ERR("ID must be 3\nHeader word is 0x%08x at file pos %d\n",header,Get_Filepos());
    return(ERROR);
  }
  if(g_frame_header.bitrate_index == 0) {
    ERR("Free bitrate format NIY!\nHeader word is 0x%08x at file pos %d\n",header,Get_Filepos());
    exit(1);
  }
  if(g_frame_header.bitrate_index == 15) {
    ERR("bitrate_index = 15 is invalid!\nHeader word is 0x%08x at file pos %d\n",header,Get_Filepos());
    return(ERROR);
  }
  if(g_frame_header.sampling_frequency == 3) {
    ERR("sampling_frequency = 3 is invalid!\n");
    ERR("Header word is 0x%08x at file pos %d\n",header,Get_Filepos());
    return(ERROR);
  }
  if(g_frame_header.layer == 0) {
    ERR("layer = 0 is invalid!\n");
    ERR("Header word is 0x%08x at file pos %d\n",header,
   Get_Filepos());
    return(ERROR);
  }
  g_frame_header.layer = 4 - g_frame_header.layer;
  /* DBG("Header         =   0x%08x\n",header); */
  return(OK);  /* Done */
}

/**Description: reads main data for layer 3 from main_data bit reservoir.
* Parameters: None
* Return value: OK or ERROR if the data contains errors.
* Author: Krister Lagerström(krister@kmlager.com) **/
static int Read_Main_L3(void){
  unsigned framesize,sideinfo_size,main_data_size,gr,ch,nch,sfb,win,slen1,slen2,nbits,part_2_start;

  /* Number of channels(1 for mono and 2 for stereo) */
  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);

  /* Calculate header audio data size */
  framesize = (144 *
    g_mpeg1_bitrates[g_frame_header.layer-1][g_frame_header.bitrate_index]) /
    g_sampling_frequency[g_frame_header.sampling_frequency] +
    g_frame_header.padding_bit;

  if(framesize > 2000) {
    ERR("framesize = %d\n",framesize);
    return(ERROR);
  }
  /* Sideinfo is 17 bytes for one channel and 32 bytes for two */
  sideinfo_size =(nch == 1 ? 17 : 32);
  /* Main data size is the rest of the frame,including ancillary data */
  main_data_size = framesize - sideinfo_size - 4 /* sync+header */;
  /* CRC is 2 bytes */
  if(g_frame_header.protection_bit == 0) main_data_size -= 2;
  /* Assemble main data buffer with data from this frame and the previous
   * two frames. main_data_begin indicates how many bytes from previous
   * frames that should be used. This buffer is later accessed by the
   * Get_Main_Bits function in the same way as the side info is.
   */
  if(Get_Main_Data(main_data_size,g_side_info.main_data_begin) != OK)
    return(ERROR); /* This could be due to not enough data in reservoir */
  for(gr = 0; gr < 2; gr++) {
    for(ch = 0; ch < nch; ch++) {
      part_2_start = Get_Main_Pos();
      /* Number of bits in the bitstream for the bands */
      slen1 = mpeg1_scalefac_sizes[g_side_info.scalefac_compress[gr][ch]][0];
      slen2 = mpeg1_scalefac_sizes[g_side_info.scalefac_compress[gr][ch]][1];
      if((g_side_info.win_switch_flag[gr][ch] != 0)&&(g_side_info.block_type[gr][ch] == 2)) {
        if(g_side_info.mixed_block_flag[gr][ch] != 0) {
          for(sfb = 0; sfb < 8; sfb++)
            g_main_data.scalefac_l[gr][ch][sfb] = Get_Main_Bits(slen1);
          for(sfb = 3; sfb < 12; sfb++) {
            nbits = (sfb < 6)?slen1:slen2;/*slen1 for band 3-5,slen2 for 6-11*/
            for(win = 0; win < 3; win++)
              g_main_data.scalefac_s[gr][ch][sfb][win]=Get_Main_Bits(nbits);
          }
        }else{
          for(sfb = 0; sfb < 12; sfb++){
            nbits = (sfb < 6)?slen1:slen2;/*slen1 for band 3-5,slen2 for 6-11*/
            for(win = 0; win < 3; win++)
              g_main_data.scalefac_s[gr][ch][sfb][win]=Get_Main_Bits(nbits);
          }
        }
      }else{ /* block_type == 0 if winswitch == 0 */
        /* Scale factor bands 0-5 */
        if((g_side_info.scfsi[ch][0] == 0) ||(gr == 0)) {
          for(sfb = 0; sfb < 6; sfb++)
            g_main_data.scalefac_l[gr][ch][sfb] = Get_Main_Bits(slen1);
        }else if((g_side_info.scfsi[ch][0] == 1) &&(gr == 1)) {
          /* Copy scalefactors from granule 0 to granule 1 */
          for(sfb = 0; sfb < 6; sfb++)
            g_main_data.scalefac_l[1][ch][sfb]=g_main_data.scalefac_l[0][ch][sfb];
        }
        /* Scale factor bands 6-10 */
        if((g_side_info.scfsi[ch][1] == 0) ||(gr == 0)) {
          for(sfb = 6; sfb < 11; sfb++)
            g_main_data.scalefac_l[gr][ch][sfb] = Get_Main_Bits(slen1);
        }else if((g_side_info.scfsi[ch][1] == 1) &&(gr == 1)) {
          /* Copy scalefactors from granule 0 to granule 1 */
          for(sfb = 6; sfb < 11; sfb++)
            g_main_data.scalefac_l[1][ch][sfb]=g_main_data.scalefac_l[0][ch][sfb];
        }
        /* Scale factor bands 11-15 */
        if((g_side_info.scfsi[ch][2] == 0) ||(gr == 0)) {
          for(sfb = 11; sfb < 16; sfb++)
            g_main_data.scalefac_l[gr][ch][sfb] = Get_Main_Bits(slen2);
        } else if((g_side_info.scfsi[ch][2] == 1) &&(gr == 1)) {
          /* Copy scalefactors from granule 0 to granule 1 */
          for(sfb = 11; sfb < 16; sfb++)
            g_main_data.scalefac_l[1][ch][sfb]=g_main_data.scalefac_l[0][ch][sfb];
        }
        /* Scale factor bands 16-20 */
        if((g_side_info.scfsi[ch][3] == 0) ||(gr == 0)) {
          for(sfb = 16; sfb < 21; sfb++)
            g_main_data.scalefac_l[gr][ch][sfb] = Get_Main_Bits(slen2);
        }else if((g_side_info.scfsi[ch][3] == 1) &&(gr == 1)) {
          /* Copy scalefactors from granule 0 to granule 1 */
          for(sfb = 16; sfb < 21; sfb++)
            g_main_data.scalefac_l[1][ch][sfb]=g_main_data.scalefac_l[0][ch][sfb];
        }
      }
      /* Read Huffman coded data. Skip stuffing bits. */
      Read_Huffman(part_2_start,gr,ch);
    } /* end for(gr... */
  } /* end for(ch... */
  /* The ancillary data is stored here,but we ignore it. */
  return(OK);  /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Error(const char *s,int e){
  (void) fwrite(s,1,strlen(s),stderr);
  exit(e);
}

/**Description: reinit decoder before playing new song,or seeking current song.
* Parameters: None
* Return value: None
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Decode_L3_Init_Song(void){
  hsynth_init = synth_init = 1;
  //g_main_data_top = 0; /* Clear bit reservoir */
}

/**Description: Does inverse modified DCT and windowing.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void IMDCT_Win(float in[18],float out[36],unsigned block_type){
  unsigned i,m,N,p;
  float tmp[12],sum,tin[18];
#ifndef IMDCT_TABLES
  static float g_imdct_win[4][36];
  static unsigned init = 1;
//TODO : move to separate init function
  if(init) { /* Setup the four(one for each block type) window vectors */
    for(i = 0; i < 36; i++)  g_imdct_win[0][i] = sin(C_PI/36 *(i + 0.5)); //0
    for(i = 0; i < 18; i++)  g_imdct_win[1][i] = sin(C_PI/36 *(i + 0.5)); //1
    for(i = 18; i < 24; i++) g_imdct_win[1][i] = 1.0;
    for(i = 24; i < 30; i++) g_imdct_win[1][i] = sin(C_PI/12 *(i + 0.5 - 18.0));
    for(i = 30; i < 36; i++) g_imdct_win[1][i] = 0.0;
    for(i = 0; i < 12; i++)  g_imdct_win[2][i] = sin(C_PI/12 *(i + 0.5)); //2
    for(i = 12; i < 36; i++) g_imdct_win[2][i] = 0.0;
    for(i = 0; i < 6; i++)   g_imdct_win[3][i] = 0.0; //3
    for(i = 6; i < 12; i++)  g_imdct_win[3][i] = sin(C_PI/12 *(i + 0.5 - 6.0));
    for(i = 12; i < 18; i++) g_imdct_win[3][i] = 1.0;
    for(i = 18; i < 36; i++) g_imdct_win[3][i] = sin(C_PI/36 *(i + 0.5));
    init = 0;
  } /* end of init */
#endif
  for(i = 0; i < 36; i++) out[i] = 0.0;
  for(i = 0; i < 18; i++) tin[i] = in[i];
  if(block_type == 2) { /* 3 short blocks */
    N = 12;
    for(i = 0; i < 3; i++) {
      for(p = 0; p < N; p++) {
        sum = 0.0;
        for(m = 0;m < N/2; m++)
#ifdef IMDCT_NTABLES
          sum += tin[i+3*m] * cos_N12[m][p];
#else
          sum += tin[i+3*m] * cos(C_PI/(2*N)*(2*p+1+N/2)*(2*m+1));
#endif
        out[6*i+p+6] += sum * g_imdct_win[block_type][p]; //TODO FIXME +=?
      }
    } /* end for(i... */
  }else{ /* block_type != 2 */
    N = 36;
    for(p = 0; p < N; p++){
      sum = 0.0;
      for(m = 0; m < N/2; m++)
#ifdef IMDCT_NTABLES
        sum += in[m] * cos_N36[m][p];
#else
        sum += in[m] * cos(C_PI/(2*N)*(2*p+1+N/2)*(2*m+1));
#endif
      out[p] = sum * g_imdct_win[block_type][p];
    }
  }
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Antialias(unsigned gr,unsigned ch){
  unsigned sb /* subband of 18 samples */,i,sblim,ui,li;
  float ub,lb;

  /* No antialiasing is done for short blocks */
  if((g_side_info.win_switch_flag[gr][ch] == 1) &&
     (g_side_info.block_type[gr][ch] == 2) &&
     (g_side_info.mixed_block_flag[gr][ch]) == 0) {
    return; /* Done */
  }
  /* Setup the limit for how many subbands to transform */
  sblim =((g_side_info.win_switch_flag[gr][ch] == 1) &&
    (g_side_info.block_type[gr][ch] == 2) &&
    (g_side_info.mixed_block_flag[gr][ch] == 1))?2:32;
  /* Do the actual antialiasing */
  for(sb = 1; sb < sblim; sb++) {
    for(i = 0; i < 8; i++) {
      li = 18*sb-1-i;
      ui = 18*sb+i;
      lb = g_main_data.is[gr][ch][li]*cs[i] - g_main_data.is[gr][ch][ui]*ca[i];
      ub = g_main_data.is[gr][ch][ui]*cs[i] + g_main_data.is[gr][ch][li]*ca[i];
      g_main_data.is[gr][ch][li] = lb;
      g_main_data.is[gr][ch][ui] = ub;
    }
  }
  return; /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Frequency_Inversion(unsigned gr,unsigned ch){
  unsigned sb,i;

  for(sb = 1; sb < 32; sb += 2) { //OPT? : for(sb = 18; sb < 576; sb += 36)
    for(i = 1; i < 18; i += 2)
      g_main_data.is[gr][ch][sb*18 + i] = -g_main_data.is[gr][ch][sb*18 + i];
  }
  return; /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Hybrid_Synthesis(unsigned gr,unsigned ch){
  unsigned sb,i,j,bt;
  float rawout[36];
  static float store[2][32][18];

  if(hsynth_init) { /* Clear stored samples vector. OPT? use memset */
    for(j = 0; j < 2; j++) {
      for(sb = 0; sb < 32; sb++) {
        for(i = 0; i < 18; i++) {
          store[j][sb][i] = 0.0;
        }
      }
    }
    hsynth_init = 0;
  } /* end if(hsynth_init) */
  for(sb = 0; sb < 32; sb++) { /* Loop through all 32 subbands */
    /* Determine blocktype for this subband */
    bt =((g_side_info.win_switch_flag[gr][ch] == 1) &&
     (g_side_info.mixed_block_flag[gr][ch] == 1) &&(sb < 2))
      ? 0 : g_side_info.block_type[gr][ch];
    /* Do the inverse modified DCT and windowing */
    IMDCT_Win(&(g_main_data.is[gr][ch][sb*18]),rawout,bt);
    for(i = 0; i < 18; i++) { /* Overlapp add with stored vector into main_data vector */
      g_main_data.is[gr][ch][sb*18 + i] = rawout[i] + store[ch][sb][i];
      store[ch][sb][i] = rawout[i + 18];
    } /* end for(i... */
  } /* end for(sb... */
  return; /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Reorder(unsigned gr,unsigned ch){
  unsigned sfreq,i,j,next_sfb,sfb,win_len,win;
  float re[576];

  sfreq = g_frame_header.sampling_frequency; /* Setup sampling freq index */
  /* Only reorder short blocks */
  if((g_side_info.win_switch_flag[gr][ch] == 1) &&
     (g_side_info.block_type[gr][ch] == 2)) { /* Short blocks */
    /* Check if the first two subbands
     *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
    sfb = (g_side_info.mixed_block_flag[gr][ch] != 0)?3:0; /* 2 longbl. sb  first */
    next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
    win_len = g_sf_band_indices[sfreq].s[sfb+1] - g_sf_band_indices[sfreq].s[sfb];
    for(i =((sfb == 0) ? 0 : 36); i < 576; /* i++ done below! */) {
      /* Check if we're into the next scalefac band */
      if(i == next_sfb) {        /* Yes */
        /* Copy reordered data back to the original vector */
        for(j = 0; j < 3*win_len; j++)
          g_main_data.is[gr][ch][3*g_sf_band_indices[sfreq].s[sfb] + j] = re[j];
        /* Check if this band is above the rzero region,if so we're done */
        if(i >= g_side_info.count1[gr][ch]) return; /* Done */
        sfb++;
        next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
        win_len = g_sf_band_indices[sfreq].s[sfb+1] - g_sf_band_indices[sfreq].s[sfb];
      } /* end if(next_sfb) */
      for(win = 0; win < 3; win++) { /* Do the actual reordering */
        for(j = 0; j < win_len; j++) {
          re[j*3 + win] = g_main_data.is[gr][ch][i];
          i++;
        } /* end for(j... */
      } /* end for(win... */
    } /* end for(i... */
    /* Copy reordered data of last band back to original vector */
    for(j = 0; j < 3*win_len; j++)
      g_main_data.is[gr][ch][3 * g_sf_band_indices[sfreq].s[12] + j] = re[j];
  } /* end else(only long blocks) */
  return; /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Requantize(unsigned gr,unsigned ch){
  unsigned sfb /* scalefac band index */,next_sfb /* frequency of next sfb */,
    sfreq,i,j,win,win_len;

  /* Setup sampling frequency index */
  sfreq = g_frame_header.sampling_frequency;
  /* Determine type of block to process */
  if((g_side_info.win_switch_flag[gr][ch] == 1) && (g_side_info.block_type[gr][ch] == 2)) { /* Short blocks */
    /* Check if the first two subbands
     *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
    if(g_side_info.mixed_block_flag[gr][ch] != 0) { /* 2 longbl. sb  first */
      /* First process the 2 long block subbands at the start */
      sfb = 0;
      next_sfb = g_sf_band_indices[sfreq].l[sfb+1];
      for(i = 0; i < 36; i++) {
        if(i == next_sfb) {
          sfb++;
          next_sfb = g_sf_band_indices[sfreq].l[sfb+1];
        } /* end if */
        Requantize_Process_Long(gr,ch,i,sfb);
      }
      /* And next the remaining,non-zero,bands which uses short blocks */
      sfb = 3;
      next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
      win_len = g_sf_band_indices[sfreq].s[sfb+1] -
        g_sf_band_indices[sfreq].s[sfb];

      for(i = 36; i < g_side_info.count1[gr][ch]; /* i++ done below! */) {
        /* Check if we're into the next scalefac band */
        if(i == next_sfb) {        /* Yes */
          sfb++;
          next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
          win_len = g_sf_band_indices[sfreq].s[sfb+1] -
            g_sf_band_indices[sfreq].s[sfb];
        } /* end if(next_sfb) */
        for(win = 0; win < 3; win++) {
          for(j = 0; j < win_len; j++) {
            Requantize_Process_Short(gr,ch,i,sfb,win);
            i++;
          } /* end for(j... */
        } /* end for(win... */

      } /* end for(i... */
    }else{ /* Only short blocks */
      sfb = 0;
      next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
      win_len = g_sf_band_indices[sfreq].s[sfb+1] -
        g_sf_band_indices[sfreq].s[sfb];
      for(i = 0; i < g_side_info.count1[gr][ch]; /* i++ done below! */) {
        /* Check if we're into the next scalefac band */
        if(i == next_sfb) {        /* Yes */
          sfb++;
          next_sfb = g_sf_band_indices[sfreq].s[sfb+1] * 3;
          win_len = g_sf_band_indices[sfreq].s[sfb+1] -
            g_sf_band_indices[sfreq].s[sfb];
        } /* end if(next_sfb) */
        for(win = 0; win < 3; win++) {
          for(j = 0; j < win_len; j++) {
            Requantize_Process_Short(gr,ch,i,sfb,win);
            i++;
          } /* end for(j... */
        } /* end for(win... */
      } /* end for(i... */
    } /* end else(only short blocks) */
  }else{ /* Only long blocks */
    sfb = 0;
    next_sfb = g_sf_band_indices[sfreq].l[sfb+1];
    for(i = 0; i < g_side_info.count1[gr][ch]; i++) {
      if(i == next_sfb) {
        sfb++;
        next_sfb = g_sf_band_indices[sfreq].l[sfb+1];
      } /* end if */
      Requantize_Process_Long(gr,ch,i,sfb);
    }
  } /* end else(only long blocks) */
  return; /* Done */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Stereo(unsigned gr){
  unsigned max_pos,i,sfreq,sfb /* scalefac band index */;
  float left,right;

  /* Do nothing if joint stereo is not enabled */
  if((g_frame_header.mode != 1)||(g_frame_header.mode_extension == 0)) return;
  /* Do Middle/Side("normal") stereo processing */
  if(g_frame_header.mode_extension & 0x2) {
    /* Determine how many frequency lines to transform */
    max_pos = g_side_info.count1[gr][!!(g_side_info.count1[gr][0] > g_side_info.count1[gr][1])];
    /* Do the actual processing */
    for(i = 0; i < max_pos; i++) {
      left =(g_main_data.is[gr][0][i] + g_main_data.is[gr][1][i])
        *(C_INV_SQRT_2);
      right =(g_main_data.is[gr][0][i] - g_main_data.is[gr][1][i])
        *(C_INV_SQRT_2);
      g_main_data.is[gr][0][i] = left;
      g_main_data.is[gr][1][i] = right;
    } /* end for(i... */
  } /* end if(ms_stereo... */
  /* Do intensity stereo processing */
  if(g_frame_header.mode_extension & 0x1) {
    /* Setup sampling frequency index */
    sfreq = g_frame_header.sampling_frequency;
    /* First band that is intensity stereo encoded is first band scale factor
     * band on or above count1 frequency line. N.B.: Intensity stereo coding is
     * only done for higher subbands, but logic is here for lower subbands. */
    /* Determine type of block to process */
    if((g_side_info.win_switch_flag[gr][0] == 1) &&
       (g_side_info.block_type[gr][0] == 2)) { /* Short blocks */
      /* Check if the first two subbands
       *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
      if(g_side_info.mixed_block_flag[gr][0] != 0) { /* 2 longbl. sb  first */
        for(sfb = 0; sfb < 8; sfb++) {/* First process 8 sfb's at start */
          /* Is this scale factor band above count1 for the right channel? */
          if(g_sf_band_indices[sfreq].l[sfb] >= g_side_info.count1[gr][1])
            Stereo_Process_Intensity_Long(gr,sfb);
        } /* end if(sfb... */
        /* And next the remaining bands which uses short blocks */
        for(sfb = 3; sfb < 12; sfb++) {
          /* Is this scale factor band above count1 for the right channel? */
          if(g_sf_band_indices[sfreq].s[sfb]*3 >= g_side_info.count1[gr][1])
            Stereo_Process_Intensity_Short(gr,sfb); /* intensity stereo processing */
        }
      }else{ /* Only short blocks */
        for(sfb = 0; sfb < 12; sfb++) {
          /* Is this scale factor band above count1 for the right channel? */
          if(g_sf_band_indices[sfreq].s[sfb]*3 >= g_side_info.count1[gr][1])
            Stereo_Process_Intensity_Short(gr,sfb); /* intensity stereo processing */
        }
      } /* end else(only short blocks) */
    }else{                        /* Only long blocks */
      for(sfb = 0; sfb < 21; sfb++) {
        /* Is this scale factor band above count1 for the right channel? */
        if(g_sf_band_indices[sfreq].l[sfb] >= g_side_info.count1[gr][1]) {
          /* Perform the intensity stereo processing */
          Stereo_Process_Intensity_Long(gr,sfb);
        }
      }
    } /* end else(only long blocks) */
  } /* end if(intensity_stereo processing) */
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void L3_Subband_Synthesis(unsigned gr,unsigned ch,unsigned outdata[576]){
  float u_vec[512],s_vec[32],sum; /* u_vec can be used insted of s_vec */
  int32_t samp;
  static unsigned init = 1;
  unsigned i,j,ss,nch;
  static float g_synth_n_win[64][32],v_vec[2 /* ch */][1024];

  /* Number of channels(1 for mono and 2 for stereo) */
  nch =(g_frame_header.mode == mpeg1_mode_single_channel) ? 1 : 2 ;
  /* Setup the n_win windowing vector and the v_vec intermediate vector */

  if(init) {
    for(i = 0; i < 64; i++) {
      for(j = 0; j < 32; j++) /*TODO: put in lookup table*/
        g_synth_n_win[i][j] = cos(((float)(16+i)*(2*j+1)) *(C_PI/64.0));
    }
    for(i = 0; i < 2; i++) /* Setup the v_vec intermediate vector */
      for(j = 0; j < 1024; j++) v_vec[i][j] = 0.0; /*TODO: memset */
    init = 0;
  } /* end if(init) */

  if(synth_init) {
    for(i = 0; i < 2; i++) /* Setup the v_vec intermediate vector */
      for(j = 0; j < 1024; j++) v_vec[i][j] = 0.0; /*TODO: memset*/
    synth_init = 0;
  } /* end if(synth_init) */

  for(ss = 0; ss < 18; ss++){ /* Loop through 18 samples in 32 subbands */
    for(i = 1023; i > 63; i--)  /* Shift up the V vector */
      v_vec[ch][i] = v_vec[ch][i-64];
    for(i = 0; i < 32; i++) /* Copy next 32 time samples to a temp vector */
      s_vec[i] =((float) g_main_data.is[gr][ch][i*18 + ss]);
    for(i = 0; i < 64; i++){ /* Matrix multiply input with n_win[][] matrix */
      sum = 0.0;
      for(j = 0; j < 32; j++) sum += g_synth_n_win[i][j] * s_vec[j];
      v_vec[ch][i] = sum;
    } /* end for(i... */
    for(i = 0; i < 8; i++) { /* Build the U vector */
      for(j = 0; j < 32; j++) { /* <<7 == *128 */
        u_vec[(i << 6) + j]      = v_vec[ch][(i << 7) + j];
        u_vec[(i << 6) + j + 32] = v_vec[ch][(i << 7) + j + 96];
      }
    } /* end for(i... */
    for(i = 0; i < 512; i++) /* Window by u_vec[i] with g_synth_dtbl[i] */
      u_vec[i] = u_vec[i] * g_synth_dtbl[i];
    for(i = 0; i < 32; i++) { /* Calc 32 samples,store in outdata vector */
      sum = 0.0;
      for(j = 0; j < 16; j++) /* sum += u_vec[j*32 + i]; */
        sum += u_vec[(j << 5) + i];
      /* sum now contains time sample 32*ss+i. Convert to 16-bit signed int */
      samp =(int32_t)(sum * 32767.0);
      if(samp > 32767) samp = 32767;
      else if(samp < -32767) samp = -32767;
      samp &= 0xffff;
      if(ch == 0) {  /* This function must be called for channel 0 first */
        /* We always run in stereo mode,& duplicate channels here for mono */
        if(nch == 1) {
          outdata[32*ss + i] =(samp << 16) |(samp);
        }else{
          outdata[32*ss + i] = samp << 16;
        }
      }else{
        outdata[32*ss + i] |= samp;
      }
    } /* end for(i... */
  } /* end for(ss... */
  return; /* Done */
}

/**Description: called by Read_Main_L3 to read Huffman coded data from bitstream.
* Parameters: None
* Return value: None. The data is stored in g_main_data.is[ch][gr][freqline].
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Read_Huffman(unsigned part_2_start,unsigned gr,unsigned ch){
  int32_t x,y,v,w;
  unsigned table_num,is_pos,bit_pos_end,sfreq;
  unsigned region_1_start,region_2_start; /* region_0_start = 0 */

  /* Check that there is any data to decode. If not,zero the array. */
  if(g_side_info.part2_3_length[gr][ch] == 0) {
    for(is_pos = 0; is_pos < 576; is_pos++)
      g_main_data.is[gr][ch][is_pos] = 0.0;
    return;
  }
  /* Calculate bit_pos_end which is the index of the last bit for this part. */
  bit_pos_end = part_2_start + g_side_info.part2_3_length[gr][ch] - 1;
  /* Determine region boundaries */
  if((g_side_info.win_switch_flag[gr][ch] == 1)&&
     (g_side_info.block_type[gr][ch] == 2)) {
    region_1_start = 36;  /* sfb[9/3]*3=36 */
    region_2_start = 576; /* No Region2 for short block case. */
  }else{
    sfreq = g_frame_header.sampling_frequency;
    region_1_start =
      g_sf_band_indices[sfreq].l[g_side_info.region0_count[gr][ch] + 1];
    region_2_start =
      g_sf_band_indices[sfreq].l[g_side_info.region0_count[gr][ch] +
        g_side_info.region1_count[gr][ch] + 2];
  }
  /* Read big_values using tables according to region_x_start */
  for(is_pos = 0; is_pos < g_side_info.big_values[gr][ch] * 2; is_pos++) {
    if(is_pos < region_1_start) {
      table_num = g_side_info.table_select[gr][ch][0];
    } else if(is_pos < region_2_start) {
      table_num = g_side_info.table_select[gr][ch][1];
    }else table_num = g_side_info.table_select[gr][ch][2];
    /* Get next Huffman coded words */
   (void) Huffman_Decode(table_num,&x,&y,&v,&w);
    /* In the big_values area there are two freq lines per Huffman word */
    g_main_data.is[gr][ch][is_pos++] = x;
    g_main_data.is[gr][ch][is_pos] = y;
  }
  /* Read small values until is_pos = 576 or we run out of huffman data */
  table_num = g_side_info.count1table_select[gr][ch] + 32;
  for(is_pos = g_side_info.big_values[gr][ch] * 2;
      (is_pos <= 572) &&(Get_Main_Pos() <= bit_pos_end); is_pos++) {
    /* Get next Huffman coded words */
   (void) Huffman_Decode(table_num,&x,&y,&v,&w);
    g_main_data.is[gr][ch][is_pos++] = v;
    if(is_pos >= 576) break;
    g_main_data.is[gr][ch][is_pos++] = w;
    if(is_pos >= 576) break;
    g_main_data.is[gr][ch][is_pos++] = x;
    if(is_pos >= 576) break;
    g_main_data.is[gr][ch][is_pos] = y;
  }
  /* Check that we didn't read past the end of this section */
  if(Get_Main_Pos() >(bit_pos_end+1)) /* Remove last words read */
    is_pos -= 4;
  /* Setup count1 which is the index of the first sample in the rzero reg. */
  g_side_info.count1[gr][ch] = is_pos;
  /* Zero out the last part if necessary */
  for(/* is_pos comes from last for-loop */; is_pos < 576; is_pos++)
    g_main_data.is[gr][ch][is_pos] = 0.0;
  /* Set the bitpos to point to the next part to read */
 (void) Set_Main_Pos(bit_pos_end+1);
  return;  /* Done */
}

/**Description: requantize sample in subband that uses long blocks.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Requantize_Process_Long(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb){
  float res,tmp1,tmp2,tmp3,sf_mult,pf_x_pt;
  static float pretab[21] = { 0,0,0,0,0,0,0,0,0,0,0,1,1,1,1,2,2,3,3,3,2 };

  sf_mult = g_side_info.scalefac_scale[gr][ch] ? 1.0 : 0.5;
  pf_x_pt = g_side_info.preflag[gr][ch] * pretab[sfb];
  tmp1 = pow(2.0,-(sf_mult *(g_main_data.scalefac_l[gr][ch][sfb] + pf_x_pt)));
  tmp2 = pow(2.0,0.25 *((int32_t) g_side_info.global_gain[gr][ch] - 210));
  if(g_main_data.is[gr][ch][is_pos] < 0.0)
    tmp3 = -Requantize_Pow_43(-g_main_data.is[gr][ch][is_pos]);
  else tmp3 = Requantize_Pow_43(g_main_data.is[gr][ch][is_pos]);
  res = g_main_data.is[gr][ch][is_pos] = tmp1 * tmp2 * tmp3;
  return; /* Done */
}

/**Description: requantize sample in subband that uses short blocks.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Requantize_Process_Short(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb,unsigned win){
  float res,tmp1,tmp2,tmp3,sf_mult;

  sf_mult = g_side_info.scalefac_scale[gr][ch] ? 1.0f : 0.5f;
  tmp1 = pow(2.0f,-(sf_mult * g_main_data.scalefac_s[gr][ch][sfb][win]));
  tmp2 = pow(2.0f,0.25f *((float) g_side_info.global_gain[gr][ch] - 210.0f -
              8.0f *(float) g_side_info.subblock_gain[gr][ch][win]));
  tmp3 =(g_main_data.is[gr][ch][is_pos] < 0.0)
    ? -Requantize_Pow_43(-g_main_data.is[gr][ch][is_pos])
    : Requantize_Pow_43(g_main_data.is[gr][ch][is_pos]);
  res = g_main_data.is[gr][ch][is_pos] = tmp1 * tmp2 * tmp3;
  return; /* Done */
}

/**Description: intensity stereo processing for entire subband with long blocks.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Stereo_Process_Intensity_Long(unsigned gr,unsigned sfb){
  unsigned i,sfreq,sfb_start,sfb_stop,is_pos;
  float is_ratio_l,is_ratio_r,left,right;

  /* Check that((is_pos[sfb]=scalefac) != 7) => no intensity stereo */
  if((is_pos = g_main_data.scalefac_l[gr][0][sfb]) != 7) {
    sfreq = g_frame_header.sampling_frequency; /* Setup sampling freq index */
    sfb_start = g_sf_band_indices[sfreq].l[sfb];
    sfb_stop = g_sf_band_indices[sfreq].l[sfb+1];
    if(is_pos == 6) { /* tan((6*PI)/12 = PI/2) needs special treatment! */
      is_ratio_l = 1.0f;
      is_ratio_r = 0.0f;
    }else{
      is_ratio_l = is_ratios[is_pos] /(1.0f + is_ratios[is_pos]);
      is_ratio_r = 1.0f /(1.0f + is_ratios[is_pos]);
    }
    /* Now decode all samples in this scale factor band */
    for(i = sfb_start; i < sfb_stop; i++) {
      left = is_ratio_l * g_main_data.is[gr][0][i];
      right = is_ratio_r * g_main_data.is[gr][0][i];
      g_main_data.is[gr][0][i] = left;
      g_main_data.is[gr][1][i] = right;
    }
  }
  return; /* Done */
} /* end Stereo_Process_Intensity_Long() */

/**Description: This function is used to perform intensity stereo processing
*              for an entire subband that uses short blocks.
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Stereo_Process_Intensity_Short(unsigned gr,unsigned sfb){
  unsigned sfb_start,sfb_stop,is_pos,is_ratio_l,is_ratio_r,i,sfreq,win,win_len;
  float left,right;

  sfreq = g_frame_header.sampling_frequency;   /* Setup sampling freq index */
  /* The window length */
  win_len = g_sf_band_indices[sfreq].s[sfb+1] - g_sf_band_indices[sfreq].s[sfb];
  /* The three windows within the band has different scalefactors */
  for(win = 0; win < 3; win++) {
    /* Check that((is_pos[sfb]=scalefac) != 7) => no intensity stereo */
    if((is_pos = g_main_data.scalefac_s[gr][0][sfb][win]) != 7) {
      sfb_start = g_sf_band_indices[sfreq].s[sfb]*3 + win_len*win;
      sfb_stop = sfb_start + win_len;
      if(is_pos == 6) { /* tan((6*PI)/12 = PI/2) needs special treatment! */
        is_ratio_l = 1.0;
        is_ratio_r = 0.0;
      }else{
        is_ratio_l = is_ratios[is_pos] /(1.0 + is_ratios[is_pos]);
        is_ratio_r = 1.0 /(1.0 + is_ratios[is_pos]);
      }
      /* Now decode all samples in this scale factor band */
      for(i = sfb_start; i < sfb_stop; i++) {
        left = is_ratio_l = g_main_data.is[gr][0][i];
        right = is_ratio_r = g_main_data.is[gr][0][i];
        g_main_data.is[gr][0][i] = left;
        g_main_data.is[gr][1][i] = right;
      }
    } /* end if(not illegal is_pos) */
  } /* end for(win... */
  return; /* Done */
} /* end Stereo_Process_Intensity_Short() */

/**Description: output audio data
* Parameters: Pointers to the samples,the number of samples
* Return value: None
* Author: Krister Lagerström(krister@kmlager.com) **/
static void audio_write(unsigned *samples,unsigned nsamples,int sample_rate){
  static int init = 0,audio,curr_sample_rate = 0;
  int tmp,dsp_speed = 44100,dsp_stereo = 2;

#ifdef OUTPUT_RAW
  audio_write_raw(samples,nsamples);
#endif /* OUTPUT_RAW */
  return;
} /* audio_write() */

/******************************************************************************
*
* Name: audio_write_raw
* Author: Krister Lagerström(krister@unidata.se)
* Description: This function is used to output raw data
* Parameters: Pointers to the samples,the number of samples
* Return value: None
* Revision History:
* Author   Date    Change
* krister  010101  Initial revision
*
******************************************************************************/
static void audio_write_raw(unsigned *samples,unsigned nsamples){
  char fname[1024];
  unsigned lo,hi;
  int i,nch;
  unsigned short s[576*2];

  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);
  for(i = 0; i < nsamples; i++) {
    if(nch == 1) {
      lo = samples[i] & 0xffff;
      s[i] = lo;
    }else{
      lo = samples[i] & 0xffff;
      hi =(samples[i] & 0xffff0000) >> 16;
      s[2*i] = hi;
      s[2*i+1] = lo;
    }
  }
  if(writeToWriter((char *) s,nsamples * 2 * nch) != nsamples * 2 * nch)
    Error("Unable to write raw data\n",-1);
  return;
} /* audio_write_raw() */
