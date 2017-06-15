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

static void audio_write(unsigned *samples,unsigned nsamples,int sample_rate);
static void audio_write_raw(unsigned *samples,unsigned nsamples);
static void Decode_L3_Init_Song(void);
static void Error(const char *s,int e);
static void L3_Requantize(unsigned gr,unsigned ch);
static void Read_Ancillary(void);
static void Requantize_Process_Long(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb);
static void Requantize_Process_Short(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb,unsigned win);

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
g_sampling_frequency[3] = { 44100 * Hz,48000 * Hz,32000 * Hz };

#ifdef POW34_ITERATE
static const float powtab34[32] = {
  0.000000f,1.000000f,2.519842f,4.326749f,6.349605f,8.549880f,10.902724f,
  13.390519f,16.000001f,18.720756f,21.544349f,24.463783f,27.473145f,30.567354f,
  33.741995f,36.993185f,40.317478f,43.711792f,47.173351f,50.699637f,54.288359f,
  57.937415f,61.644873f,65.408949f,69.227988f,73.100453f,77.024908f,81.000011f,
  85.024502f,89.097200f,93.216988f,97.382814f
}
#endif
  ;

unsigned synth_init = 1;

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

t_mpeg1_header    g_frame_header;
t_mpeg1_side_info g_side_info;  /* < 100 words */
t_mpeg1_main_data g_main_data;  /* Large static data(~2500 words) */

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
  synth_init = 1;
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

/**Description: called by Read_Main_L3 to read Huffman coded data from bitstream.
* Parameters: None
* Return value: None. The data is stored in g_main_data.is[ch][gr][freqline].
* Author: Krister Lagerström(krister@kmlager.com) **/
void Read_Huffman(unsigned part_2_start,unsigned gr,unsigned ch){
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
