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
static void Error(const char *s,int e);

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

unsigned synth_init = 1;

t_mpeg1_header    g_frame_header;

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

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Error(const char *s,int e){
  (void) fwrite(s,1,strlen(s),stderr);
  exit(e);
}
